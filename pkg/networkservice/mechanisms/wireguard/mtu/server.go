// Copyright (c) 2021-2022 Cisco and/or its affiliates.
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
	"net"

	"git.fd.io/govpp.git/api"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/wireguard"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/tunnelmtu"
)

type mtuServer struct {
	vppConn  api.Connection
	tunnelIP net.IP
}

// NewServer - server chain element to manage wireguard MTU
func NewServer(vppConn api.Connection, tunnelIP net.IP) networkservice.NetworkServiceServer {
	return &mtuServer{
		vppConn:  vppConn,
		tunnelIP: tunnelIP,
	}
}

func (m *mtuServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if mechanism := wireguard.ToMechanism(request.GetConnection().GetMechanism()); mechanism != nil {
		tunnelMTU, ok := tunnelmtu.Load(ctx, metadata.IsClient(m))
		if !ok {
			return nil, errors.New("tunnel MTU is required")
		}

		// If the clients MTU is zero or larger than the mtu for the local end of the tunnel, use the mtu from the local end of the tunnel
		mtu := tunnelMTU - overhead(m.tunnelIP.To4() == nil)
		if mechanism.MTU() > mtu || mechanism.MTU() == 0 {
			mechanism.SetMTU(mtu)
		}
	}
	return next.Server(ctx).Request(ctx, request)
}

func (m *mtuServer) Close(ctx context.Context, conn *networkservice.Connection) (*emptypb.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
