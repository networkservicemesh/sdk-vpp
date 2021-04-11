// Copyright (c) 2020-2021 Cisco and/or its affiliates.
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

package l2xconnect

import (
	"context"

	"git.fd.io/govpp.git/api"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type l2XconnectServer struct {
	vppConn api.Connection
}

// NewServer returns a Server chain element that will cross connect a client and server vpp interface (if present)
func NewServer(vppConn api.Connection) networkservice.NetworkServiceServer {
	return &l2XconnectServer{
		vppConn: vppConn,
	}
}

func (v *l2XconnectServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if request.GetConnection().GetPayload() != payload.Ethernet {
		return next.Server(ctx).Request(ctx, request)
	}
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	if err := addDel(ctx, v.vppConn, true); err != nil {
		_, _ = v.Close(ctx, conn)
		return nil, err
	}
	return conn, nil
}

func (v *l2XconnectServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	if conn.GetPayload() != payload.Ethernet {
		return next.Server(ctx).Close(ctx, conn)
	}
	rv, err := next.Server(ctx).Close(ctx, conn)
	if err != nil {
		return nil, err
	}
	if err := addDel(ctx, v.vppConn, false); err != nil {
		return nil, err
	}
	return rv, nil
}
