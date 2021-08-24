// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/loopback"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type vrfServer struct {
	vppConn api.Connection

	tablesV4 vrfMap
	tablesV6 vrfMap
}

// NewServer creates a NetworkServiceServer chain element to create the ip table in vpp
func NewServer(vppConn api.Connection) networkservice.NetworkServiceServer {
	return &vrfServer{
		vppConn:   vppConn,
		tablesV4: vrfMap{
			entries: make(map[string]*vrfInfo),
		},
		tablesV6: vrfMap{
			entries: make(map[string]*vrfInfo),
		},
	}
}

func (v *vrfServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	attached := false
	conn := request.GetConnection()
	ipNets := conn.GetContext().GetIpContext().GetDstIPRoutes()
	for	idx := range ipNets {
		isIPv6 := ipNets[idx].GetPrefixIPNet().IP.To4() == nil
		t := &v.tablesV4
		if isIPv6 {
			t = &v.tablesV6
		}
		if _, ok := Load(ctx, metadata.IsClient(v), isIPv6); !ok {
			vrfId, err := create(ctx, conn, v.vppConn, t, isIPv6, metadata.IsClient(v));
			if err != nil {
				return nil, err
			}
			Store(ctx, metadata.IsClient(v), isIPv6, vrfId)
		} else {
			attached = true
		}
	}

	conn, err := next.Server(ctx).Request(ctx, request)

	if !attached {
		/* Attach loopback to vrf-tables */
		if swIfIndex, ok := loopback.Load(ctx, metadata.IsClient(v)); ok {
			if err := attach(ctx, v.vppConn, swIfIndex, metadata.IsClient(v)); err != nil {
				return nil, err
			}
		}
		/* Attach interface to vrf-tables */
		if swIfIndex, ok := ifindex.Load(ctx, metadata.IsClient(v)); ok {
			if err := attach(ctx, v.vppConn, swIfIndex, metadata.IsClient(v)); err != nil {
				return nil, err
			}
		}
	}
	return conn, err
}

func (v *vrfServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	/* Check ipv4 and ipv6 */
	for _, isIpV6 := range[]bool{false, true} {
		t := &v.tablesV4
		if isIpV6 {
			t = &v.tablesV6
		}
		del(ctx, conn, v.vppConn, t, isIpV6, metadata.IsClient(v))
	}

	return next.Server(ctx).Close(ctx, conn)
}
