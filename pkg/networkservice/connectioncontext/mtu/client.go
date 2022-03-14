// Copyright (c) 2021 Cisco and/or its affiliates.
//
// Copyright (c) 2022 Nordix Foundation.
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

package mtu

import (
	"context"

	"git.fd.io/govpp.git/api"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vlan"
	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
)

type mtuClient struct {
	vppConn api.Connection
}

// NewClient creates a NetworkServiceClient chain element to set the mtu on a vpp interface
// It sets the mtu on the *vpp* side of an interface leaving the
// Endpoint.
//                                         Endpoint
//                              +---------------------------+
//                              |                           |
//                              |                           |
//                              |                           |
//                              |                           |
//                              |                           |
//                              |                           |
//                              |                           |
//                              |            mtu.NewClient()+-------------------+
//                              |                           |
//                              |                           |
//                              |                           |
//                              |                           |
//                              |                           |
//                              |                           |
//                              |                           |
//                              +---------------------------+
//
func NewClient(vppConn api.Connection) networkservice.NetworkServiceClient {
	return &mtuClient{
		vppConn: vppConn,
	}
}

func (m *mtuClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	setConnContextMTU(request)

	postponeCtxFunc := postpone.ContextWithValues(ctx)

	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	if vlan.ToMechanism(conn.GetMechanism()) != nil {
		// No need to set the MTU since it is fetched from this interface
		return conn, nil
	}

	if conn.GetPayload() == payload.Ethernet {
		if err := setVPPL2MTU(ctx, conn, m.vppConn, metadata.IsClient(m)); err != nil {
			if closeErr := m.closeOnFailure(postponeCtxFunc, conn, opts); closeErr != nil {
				err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
			}
			return nil, err
		}
	}

	if conn.GetPayload() == payload.IP {
		if err := setVPPL3MTU(ctx, conn, m.vppConn, metadata.IsClient(m)); err != nil {
			if closeErr := m.closeOnFailure(postponeCtxFunc, conn, opts); closeErr != nil {
				err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
			}
			return nil, err
		}
	}

	return conn, nil
}

func (m *mtuClient) closeOnFailure(postponeCtxFunc func() (context.Context, context.CancelFunc), conn *networkservice.Connection, opts []grpc.CallOption) error {
	closeCtx, cancelClose := postponeCtxFunc()
	defer cancelClose()

	_, err := m.Close(closeCtx, conn, opts...)

	return err
}

func (m *mtuClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
