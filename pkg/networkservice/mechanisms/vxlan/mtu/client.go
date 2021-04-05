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
	"net"
	"sync"

	"git.fd.io/govpp.git/api"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vxlan"
)

type mtuClient struct {
	vppConn  api.Connection
	tunnelIP net.IP
	mtu      uint32
	initOnce sync.Once
	err      error
}

// NewClient - returns client chain element to manage vxlan MTU
func NewClient(vppConn api.Connection, tunnelIP net.IP) networkservice.NetworkServiceClient {
	return &mtuClient{
		vppConn:  vppConn,
		tunnelIP: tunnelIP,
	}
}

func (m *mtuClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	if err := m.init(ctx); err != nil {
		return nil, err
	}
	if mechanism := vxlan.ToMechanism(request.GetConnection().GetMechanism()); mechanism != nil && (mechanism.MTU() == 0 || mechanism.MTU() > m.mtu) {
		mechanism.SetMTU(m.mtu)
	}
	for _, mech := range request.GetMechanismPreferences() {
		if mechanism := vxlan.ToMechanism(mech); mechanism != nil && (mechanism.MTU() == 0 || mechanism.MTU() > m.mtu) {
			mechanism.SetMTU(m.mtu)
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

func (m *mtuClient) init(ctx context.Context) error {
	m.initOnce.Do(func() {
		m.mtu, m.err = setMTU(ctx, m.vppConn, m.tunnelIP)
	})
	return m.err
}
