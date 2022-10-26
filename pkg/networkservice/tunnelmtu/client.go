// Copyright (c) 2022 Cisco and/or its affiliates.
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

package tunnelmtu

import (
	"context"
	"net"
	"sync"
	"sync/atomic"

	"git.fd.io/govpp.git/api"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type mtuClient struct {
	vppConn  api.Connection
	tunnelIP net.IP
	mtu      uint32

	inited    uint32
	initMutex sync.Mutex
}

// NewClient - returns client chain element that calculates MTU of the tunnel interface
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
	Store(ctx, metadata.IsClient(m), m.mtu)
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (m *mtuClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}

func (m *mtuClient) init(ctx context.Context) error {
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
