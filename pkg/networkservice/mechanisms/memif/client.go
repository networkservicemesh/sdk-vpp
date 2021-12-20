// Copyright (c) 2020-2021 Cisco and/or its affiliates.
//
// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

//+build linux

package memif

import (
	"context"

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
)

type memifClient struct {
	vppConn     *vppConnection
	changeNetNs bool
	nsInfo      NetNSInfo
}

// NewClient provides a NetworkServiceClient chain elements that support the memif Mechanism
func NewClient(vppConn api.Connection, options ...Option) networkservice.NetworkServiceClient {
	opts := &memifOptions{}
	for _, o := range options {
		o(opts)
	}

	return chain.NewNetworkServiceClient(
		&memifClient{
			vppConn: &vppConnection{
				isExternal: opts.isVPPExternal,
				Connection: vppConn,
			},
			changeNetNs: opts.changeNetNS,
			nsInfo:      newNetNSInfo(),
		},
	)
}

func mechanismsContain(list []*networkservice.Mechanism, t string) bool {
	for _, m := range list {
		if m.Type == t {
			return true
		}
	}
	return false
}

func (m *memifClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	if !mechanismsContain(request.MechanismPreferences, memif.MECHANISM) {
		mechanism := memif.ToMechanism(memif.NewAbstract(m.nsInfo.netNSPath))
		if m.changeNetNs {
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
