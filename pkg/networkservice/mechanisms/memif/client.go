// Copyright (c) 2020-2022 Cisco and/or its affiliates.
//
// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
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
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/memif"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/memif/memifproxy"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/memif/memifrxmode"
)

type memifClient struct {
	vppConn     api.Connection
	changeNetNS bool
	nsInfo      NetNSInfo
}

// NewClient provides a NetworkServiceClient chain elements that support the memif Mechanism
func NewClient(chainCtx context.Context, vppConn Connection, options ...Option) networkservice.NetworkServiceClient {
	opts := &memifOptions{}
	for _, o := range options {
		o(opts)
	}

	return chain.NewNetworkServiceClient(
		memifrxmode.NewClient(chainCtx, vppConn),
		&memifClient{
			vppConn:     vppConn,
			changeNetNS: opts.changeNetNS,
			nsInfo:      newNetNSInfo(),
		},
	)
}

func (m *memifClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	if !m.updateMechanismPreferences(request) {
		mechanism := memif.ToMechanism(memif.NewAbstract(m.nsInfo.netNSPath))
		if m.changeNetNS {
			mechanism.SetNetNSURL("")
		}
		request.MechanismPreferences = append(request.MechanismPreferences, mechanism.Mechanism)
	}

	postponeCtxFunc := postpone.ContextWithValues(ctx)

	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	// If direct memif case store mechanism to metadata and return.
	if info, ok := memifproxy.LoadInfo(ctx); ok {
		if mechanism := memif.ToMechanism(conn.GetMechanism()); mechanism != nil && ok {
			info.NSURL = mechanism.GetNetNSURL()
			info.SocketFile = mechanism.GetSocketFilename()
			return conn, nil
		}
	}

	if err = create(ctx, conn, m.vppConn, metadata.IsClient(m), m.nsInfo.netNS); err != nil {
		closeCtx, cancelClose := postponeCtxFunc()
		defer cancelClose()

		if _, closeErr := m.Close(closeCtx, conn, opts...); closeErr != nil {
			err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
		}

		return nil, err
	}

	return conn, nil
}

func (m *memifClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	_ = del(ctx, conn, m.vppConn, metadata.IsClient(m))
	return next.Client(ctx).Close(ctx, conn, opts...)
}

// updateMechanismPreferences returns true if MechanismPreferences has updated
func (m *memifClient) updateMechanismPreferences(request *networkservice.NetworkServiceRequest) bool {
	var updated = false
	for _, p := range request.GetRequestMechanismPreferences() {
		if mechanism := memif.ToMechanism(p); mechanism != nil {
			mechanism.SetNetNSURL((&url.URL{Scheme: memif.FileScheme, Path: m.nsInfo.netNSPath}).String())
			if m.changeNetNS {
				mechanism.SetNetNSURL("")
			}
			updated = true
		}
	}
	return updated
}
