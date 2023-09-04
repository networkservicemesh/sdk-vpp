// Copyright (c) 2021-2023 Cisco and/or its affiliates.
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
	"sync/atomic"

	"go.fd.io/govpp/api"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vxlan"
)

type mtuServer struct {
	vppConn  api.Connection
	tunnelIP net.IP
	mtu      uint32

	inited    uint32
	initMutex sync.Mutex
}

// NewServer - server chain element to manage vxlan MTU
func NewServer(vppConn api.Connection, tunnelIP net.IP) networkservice.NetworkServiceServer {
	return &mtuServer{
		vppConn:  vppConn,
		tunnelIP: tunnelIP,
	}
}

func (m *mtuServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if err := m.init(ctx); err != nil {
		return nil, err
	}
	if mechanism := vxlan.ToMechanism(request.GetConnection().GetMechanism()); mechanism != nil {
		if err := m.init(ctx); err != nil {
			return nil, err
		}
		// If the clients MTU is zero or larger than the mtu for the local end of the tunnel, use the the mtu from the local end of the tunnel
		if mechanism.MTU() > m.mtu || mechanism.MTU() == 0 {
			mechanism.SetMTU(m.mtu)
		}
		// If the ConnectionContext's MTU is zero or larger than the MTU for the tunnel, set the ConnectionContexts MTU to the MTU for the tunnel
		if request.GetConnection().GetContext().GetMTU() > mechanism.MTU() || request.GetConnection().GetContext().GetMTU() == 0 {
			if request.GetConnection() == nil {
				request.Connection = &networkservice.Connection{}
			}
			if request.GetConnection().GetContext() == nil {
				request.GetConnection().Context = &networkservice.ConnectionContext{}
			}
			request.GetConnection().GetContext().MTU = mechanism.MTU()
		}
	}
	return next.Server(ctx).Request(ctx, request)
}

func (m *mtuServer) Close(ctx context.Context, conn *networkservice.Connection) (*emptypb.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}

func (m *mtuServer) init(ctx context.Context) error {
	if atomic.LoadUint32(&m.inited) > 0 {
		return nil
	}
	m.initMutex.Lock()
	defer m.initMutex.Unlock()
	if atomic.LoadUint32(&m.inited) > 0 {
		return nil
	}

	var err error
	m.mtu, err = getMTU(ctx, m.vppConn, m.tunnelIP)
	if err == nil {
		atomic.StoreUint32(&m.inited, 1)
	}
	return err
}
