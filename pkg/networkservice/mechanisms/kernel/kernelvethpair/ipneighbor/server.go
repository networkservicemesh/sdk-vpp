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

// +build linux

package ipneighbor

import (
	"context"

	"git.fd.io/govpp.git/api"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type ipneighborServer struct {
	vppConn api.Connection
}

// NewServer - creates new ipneigbor server chain element to correct for the L2 nature of vethpairs when used for payload.IP
func NewServer(vppConn api.Connection) networkservice.NetworkServiceServer {
	return &ipneighborServer{
		vppConn: vppConn,
	}
}

func (i *ipneighborServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if request.GetConnection().GetPayload() != payload.IP {
		return next.Server(ctx).Request(ctx, request)
	}
	if err := addDel(ctx, request.GetConnection(), i.vppConn, metadata.IsClient(i), true); err != nil {
		return nil, err
	}
	return next.Server(ctx).Request(ctx, request)
}

func (i *ipneighborServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	if conn.GetPayload() != payload.IP {
		return next.Server(ctx).Close(ctx, conn)
	}
	_ = addDel(ctx, conn, i.vppConn, metadata.IsClient(i), false)
	return next.Server(ctx).Close(ctx, conn)
}
