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
	"errors"
	"net"

	"git.fd.io/govpp.git/api"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/wireguard"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/tunnelmtu"
)

type mtuClient struct {
	vppConn  api.Connection
	tunnelIP net.IP
}

// NewClient - returns client chain element to manage wireguard MTU
func NewClient(vppConn api.Connection, tunnelIP net.IP) networkservice.NetworkServiceClient {
	return &mtuClient{
		vppConn:  vppConn,
		tunnelIP: tunnelIP,
	}
}

func (m *mtuClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	tunnelMTU, ok := tunnelmtu.Load(ctx, metadata.IsClient(m))
	if !ok {
		return nil, errors.New("tunnel MTU is required")
	}

	mtu := tunnelMTU - overhead(m.tunnelIP.To4() == nil)
	if mechanism := wireguard.ToMechanism(request.GetConnection().GetMechanism()); mechanism != nil && (mechanism.MTU() == 0 || mechanism.MTU() > mtu) {
		mechanism.SetMTU(mtu)
	}
	for _, mech := range request.GetMechanismPreferences() {
		if mechanism := wireguard.ToMechanism(mech); mechanism != nil && (mechanism.MTU() == 0 || mechanism.MTU() > mtu) {
			mechanism.SetMTU(mtu)
		}
	}
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (m *mtuClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
