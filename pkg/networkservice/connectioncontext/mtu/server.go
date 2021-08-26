// Copyright (c) 2021 Cisco and/or its affiliates.
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

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
)

type mtuServer struct {
	vppConn api.Connection
}

// NewServer creates a NetworkServiceServer chain element to set the MTU on a vpp interface
// It sets the MTU on the *vpp* side of an interface plugged into the
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
//          +-------------------+ mtu.NewServer()           |
//                              |                           |
//                              |                           |
//                              |                           |
//                              |                           |
//                              |                           |
//                              |                           |
//                              |                           |
//                              +---------------------------+
//
func NewServer(vppConn api.Connection) networkservice.NetworkServiceServer {
	return &mtuServer{
		vppConn: vppConn,
	}
}

func (m *mtuServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	setConnContextMTU(request)

	postponeCtxFunc := postpone.ContextWithValues(ctx)

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	if conn.GetPayload() == payload.Ethernet {
		if err := setVPPL2MTU(ctx, conn, m.vppConn, metadata.IsClient(m)); err != nil {
			if closeErr := m.closeOnFailure(postponeCtxFunc, conn); closeErr != nil {
				err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
			}
			return nil, err
		}
	}

	if conn.GetPayload() == payload.IP {
		if err := setVPPL3MTU(ctx, conn, m.vppConn, metadata.IsClient(m)); err != nil {
			if closeErr := m.closeOnFailure(postponeCtxFunc, conn); closeErr != nil {
				err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
			}
			return nil, err
		}
	}

	return conn, nil
}

func (m *mtuServer) closeOnFailure(postponeCtxFunc func() (context.Context, context.CancelFunc), conn *networkservice.Connection) error {
	closeCtx, cancelClose := postponeCtxFunc()
	defer cancelClose()

	_, err := m.Close(closeCtx, conn)

	return err
}

func (m *mtuServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
