// Copyright (c) 2020-2021 Cisco and/or its affiliates.
//
// Copyright (c) 2021 Nordix Foundation.
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

package vxlan

import (
	"context"
	"net"

	"git.fd.io/govpp.git/api"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"

	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/vxlan/mtu"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/vxlan/vni"
)

type vxlanServer struct {
	vppConn api.Connection
}

// NewServer - returns a new server for the vxlan remote mechanism
func NewServer(vppConn api.Connection, tunnelIP net.IP) networkservice.NetworkServiceServer {
	return chain.NewNetworkServiceServer(
		vni.NewServer(tunnelIP),
		mtu.NewServer(vppConn, tunnelIP),
		&vxlanServer{
			vppConn: vppConn,
		},
	)
}

func (v *vxlanServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if request.GetConnection().GetPayload() != payload.Ethernet {
		return next.Server(ctx).Request(ctx, request)
	}
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	if err := addDel(ctx, conn, v.vppConn, true, metadata.IsClient(v)); err != nil {
		_, _ = v.Close(ctx, conn)
		return nil, err
	}
	if err := addDel(ctx, request.GetConnection(), v.vppConn, true, metadata.IsClient(v)); err != nil {
		return nil, err
	}
	return conn, nil
}

func (v *vxlanServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	if conn.GetPayload() != payload.Ethernet {
		return next.Server(ctx).Close(ctx, conn)
	}
	if err := addDel(ctx, conn, v.vppConn, false, false); err != nil {
		return nil, err
	}
	return next.Server(ctx).Close(ctx, conn)
}
