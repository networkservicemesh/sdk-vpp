// Copyright (c) 2020 Cisco and/or its affiliates.
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

package routes

import (
	"context"

	"git.fd.io/govpp.git/api"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type routesServer struct {
	vppConn api.Connection
}

// NewServer creates a NetworkServiceServer chain element to set the ip address on a vpp interface
// It sets the IP Address on the *vpp* side of an interface plugged into the
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
//          +-------------------+ ipaddress.NewServer()     |
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
	return &routesServer{
		vppConn: vppConn,
	}
}

func (r *routesServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if err := addDel(ctx, request.GetConnection(), r.vppConn, metadata.IsClient(r), true); err != nil {
		return nil, err
	}
	return next.Server(ctx).Request(ctx, request)
}

func (r *routesServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	if err := addDel(ctx, conn, r.vppConn, metadata.IsClient(r), false); err != nil {
		return nil, err
	}
	return next.Server(ctx).Close(ctx, conn)
}
