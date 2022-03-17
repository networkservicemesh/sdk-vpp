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
	vppConn         api.Connection
	loadInterfaceFn ifindex.LoadInterfaceFn

	v4 *vrfMap
	v6 *vrfMap
}

// NewServer creates a NetworkServiceServer chain element to create the ip table in vpp
func NewServer(vppConn api.Connection, opts ...Option) networkservice.NetworkServiceServer {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	return &vrfServer{
		vppConn: vppConn,
		v4:      newMap(),
		v6:      newMap(),
	}
}

func (v *vrfServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	intAttached := false
	lbAttached := false
	networkService := request.GetConnection().NetworkService
	for _, isIPv6 := range []bool{false, true} {
		t := v.v4
		if isIPv6 {
			t = v.v6
		}
		if _, ok := Load(ctx, metadata.IsClient(v), isIPv6); !ok {
			vrfID, loaded, err := loadOrCreate(ctx, v.vppConn, networkService, t, isIPv6)
			if err != nil {
				return nil, err
			}
			Store(ctx, metadata.IsClient(v), isIPv6, vrfID)
			lbAttached = loaded
		} else {
			intAttached = true
			lbAttached = true
		}
	}
	postponeCtxFunc := postpone.ContextWithValues(ctx)
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		v.delV46(ctx, networkService)
		return conn, err
	}

	if !lbAttached && v.loadInterfaceFn == nil {
		/* Attach iface to vrf-tables */
		if swIfIndex, ok := v.loadInterfaceFn(ctx, metadata.IsClient(v)); ok {
			if err = attach(ctx, v.vppConn, swIfIndex, metadata.IsClient(v)); err != nil {
				closeCtx, cancelClose := postponeCtxFunc()
				defer cancelClose()
				v.delV46(ctx, conn.NetworkService)
				if _, closeErr := next.Server(closeCtx).Close(closeCtx, conn); closeErr != nil {
					err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
				}
				return nil, err
			}
		}
	}
	if !intAttached {
		/* Attach interface to vrf-tables */
		if swIfIndex, ok := ifindex.Load(ctx, metadata.IsClient(v)); ok {
			if attachErr := attach(ctx, v.vppConn, swIfIndex, metadata.IsClient(v)); attachErr != nil {
				return nil, attachErr
			}
		}
	}
	return conn, err
}

func (v *vrfServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	_, err := next.Server(ctx).Close(ctx, conn)

	v.delV46(ctx, conn.NetworkService)
	return &empty.Empty{}, err
}

/* Delete from ipv4 and ipv6 vrfs */
func (v *vrfServer) delV46(ctx context.Context, networkService string) {
	del(ctx, v.vppConn, networkService, v.v4, false, metadata.IsClient(v))
	del(ctx, v.vppConn, networkService, v.v6, true, metadata.IsClient(v))
}
