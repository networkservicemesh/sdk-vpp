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

package vrf

import (
	"context"

	"git.fd.io/govpp.git/api"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
)

type vrfServer struct {
	vppConn api.Connection
	loadFn  ifindex.LoadInterfaceFn
	m       *Map
}

// NewServer creates a NetworkServiceServer chain element to create the ip table in vpp
func NewServer(vppConn api.Connection, opts ...Option) networkservice.NetworkServiceServer {
	o := &options{
		m:      NewMap(),
		loadFn: ifindex.Load,
	}
	for _, opt := range opts {
		opt(o)
	}

	return &vrfServer{
		vppConn: vppConn,
		loadFn:  o.loadFn,
		m:       o.m,
	}
}

func (v *vrfServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	var loadIfaces = []ifindex.LoadInterfaceFn{v.loadFn, ifindex.Load}
	var networkService = request.GetConnection().GetNetworkService()

	for _, isIPv6 := range []bool{false, true} {
		t := v.m.ipv4
		if isIPv6 {
			t = v.m.ipv6
		}
		if _, ok := Load(ctx, metadata.IsClient(v), isIPv6); !ok {
			vrfID, _, err := create(ctx, v.vppConn, networkService, t, isIPv6)
			if err != nil {
				return nil, err
			}
			Store(ctx, metadata.IsClient(v), isIPv6, vrfID)
		} else {
			loadIfaces = nil
		}
	}
	postponeCtxFunc := postpone.ContextWithValues(ctx)
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		delV46(ctx, v.vppConn, v.m, conn.GetNetworkService(), metadata.IsClient(v))
		return conn, err
	}

	for _, loadFn := range loadIfaces {
		if swIfIndex, ok := loadFn(ctx, metadata.IsClient(v)); ok {
			if attachErr := attach(ctx, v.vppConn, networkService, v.m, swIfIndex, metadata.IsClient(v)); attachErr != nil {
				closeCtx, cancelClose := postponeCtxFunc()
				defer cancelClose()
				delV46(ctx, v.vppConn, v.m, conn.GetNetworkService(), metadata.IsClient(v))

				if _, closeErr := next.Server(closeCtx).Close(closeCtx, conn); closeErr != nil {
					attachErr = errors.Wrapf(attachErr, "connection closed with error: %s", closeErr.Error())
				}
				return nil, attachErr
			}
		}
	}

	return conn, err
}

func (v *vrfServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	_, err := next.Server(ctx).Close(ctx, conn)

	delV46(ctx, v.vppConn, v.m, conn.GetNetworkService(), metadata.IsClient(v))
	return &empty.Empty{}, err
}
