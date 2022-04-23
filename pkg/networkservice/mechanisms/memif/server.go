// Copyright (c) 2020-2022 Cisco and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build linux
// +build linux

package memif

import (
	"context"
	"net/url"

	"git.fd.io/govpp.git/api"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/memif"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/memif/memifproxy"
)

type memifServer struct {
	vppConn     api.Connection
	changeNetNS bool
	nsInfo      NetNSInfo
}

// NewServer provides a NetworkServiceServer chain elements that support the memif Mechanism
func NewServer(chainCtx context.Context, vppConn api.Connection, options ...Option) networkservice.NetworkServiceServer {
	opts := new(memifOptions)
	for _, o := range options {
		o(opts)
	}

	memifProxyServer := null.NewServer()
	if opts.directMemifEnabled {
		memifProxyServer = memifproxy.NewServer(chainCtx)
	}

	return chain.NewNetworkServiceServer(
		memifProxyServer,
		&memifServer{
			vppConn:     vppConn,
			changeNetNS: opts.changeNetNS,
			nsInfo:      newNetNSInfo(),
		},
	)
}

func (m *memifServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	postponeCtxFunc := postpone.ContextWithValues(ctx)

	if mechanism := memif.ToMechanism(request.GetConnection().GetMechanism()); mechanism != nil && !m.changeNetNS {
		mechanism.SetNetNSURL((&url.URL{Scheme: memif.FileScheme, Path: m.nsInfo.netNSPath}).String())
	}

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	// In direct memif case do nothing.
	if info, ok := memifproxy.LoadInfo(ctx); ok && info.SocketFile != "" {
		return conn, nil
	}

	if err = create(ctx, conn, m.vppConn, metadata.IsClient(m), m.nsInfo.netNS); err != nil {
		closeCtx, cancelClose := postponeCtxFunc()
		defer cancelClose()

		if _, closeErr := m.Close(closeCtx, conn); closeErr != nil {
			err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
		}

		return nil, err
	}

	return conn, nil
}

func (m *memifServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	_ = del(ctx, conn, m.vppConn, metadata.IsClient(m))
	return next.Server(ctx).Close(ctx, conn)
}
