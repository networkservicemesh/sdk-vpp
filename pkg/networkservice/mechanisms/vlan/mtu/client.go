// Copyright (c) 2022 Nordix Foundation.
//
// Copyright (c) 2023 Cisco and/or its affiliates.
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

	"go.fd.io/govpp/api"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vlan"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
)

type mtuClient struct {
	vppConn     api.Connection
	mtu         mtuMap
	deviceNames map[string]string
}

// NewClient - returns client chain element to manage vlan MTU
func NewClient(vppConn api.Connection, deviceNames map[string]string) networkservice.NetworkServiceClient {
	return &mtuClient{
		vppConn:     vppConn,
		deviceNames: deviceNames,
	}
}

func (m *mtuClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	postponeCtxFunc := postpone.ContextWithValues(ctx)

	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}
	_, ok := ifindex.Load(ctx, metadata.IsClient(m))
	if !ok {
		return conn, nil
	}
	if mechanism := vlan.ToMechanism(conn.GetMechanism()); mechanism != nil {
		via := conn.GetLabels()[viaLabel]
		hostIFName, ok := m.deviceNames[via]
		if !ok {
			return nil, errors.New("can not find device name for via label")
		}
		localMtu, loaded := m.mtu.Load(hostIFName)
		if !loaded {
			localMtu, err = getMTU(ctx, m.vppConn, hostIFName)
			if err != nil {
				closeCtx, cancelClose := postponeCtxFunc()
				defer cancelClose()
				if _, closeErr := m.Close(closeCtx, conn); closeErr != nil {
					err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
				}
				return nil, err
			}
			m.mtu.Store(hostIFName, localMtu)
		}
		if localMtu > 0 && (conn.GetContext().GetMTU() > localMtu || conn.GetContext().GetMTU() == 0) {
			if conn.GetContext() == nil {
				conn.Context = &networkservice.ConnectionContext{}
			}
			conn.GetContext().MTU = localMtu
		}
	}
	return conn, nil
}

func (m *mtuClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
