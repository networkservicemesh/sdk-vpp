// Copyright (c) 2020 Cisco and/or its affiliates.
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

package routes

import (
	"context"
	"net"
	"time"

	"git.fd.io/govpp.git/api"
	"github.com/edwarnicke/govpp/binapi/fib_types"
	"github.com/edwarnicke/govpp/binapi/interface_types"
	"github.com/edwarnicke/govpp/binapi/ip"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/types"
)

func addDel(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection, isClient, isAdd bool) error {
	swIfIndex, ok := ifindex.Load(ctx, isClient)
	if !ok {
		return nil
	}
	from := conn.GetContext().GetIpContext().GetDstIPNet()
	to := conn.GetContext().GetIpContext().GetSrcIPNet()
	if isClient {
		from = conn.GetContext().GetIpContext().GetSrcIPNet()
		to = conn.GetContext().GetIpContext().GetDstIPNet()
	}
	routes := conn.GetContext().GetIpContext().GetDstRoutes()
	if isClient {
		routes = conn.GetContext().GetIpContext().GetSrcRoutes()
	}
	for _, route := range routes {
		if err := routeAddDel(ctx, vppConn, swIfIndex, isAdd, route.GetPrefixIPNet(), to); err != nil {
			return err
		}
	}
	if to != nil && !to.Contains(from.IP) {
		if err := routeAddDel(ctx, vppConn, swIfIndex, isAdd, to, nil); err != nil {
			return err
		}
	}
	return nil
}

func routeAddDel(ctx context.Context, vppConn api.Connection, swIfIndex interface_types.InterfaceIndex, isAdd bool, prefix, gw *net.IPNet) error {
	if prefix == nil {
		return errors.New("route prefix must not be nil")
	}
	route := route(prefix, swIfIndex, gw)
	now := time.Now()
	if _, err := ip.NewServiceClient(vppConn).IPRouteAddDel(ctx, &ip.IPRouteAddDel{
		IsAdd:       isAdd,
		IsMultipath: true,
		Route:       route,
	}); err != nil {
		return errors.WithStack(err)
	}
	log.FromContext(ctx).
		WithField("swIfIndex", swIfIndex).
		WithField("prefix", prefix).
		WithField("isAdd", isAdd).
		WithField("type", isAdd).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "IPRouteAddDel").Debug("completed")
	return nil
}

func route(dst *net.IPNet, via interface_types.InterfaceIndex, nh *net.IPNet) ip.IPRoute {
	route := ip.IPRoute{
		StatsIndex: 0,
		Prefix:     types.ToVppPrefix(dst),
		NPaths:     1,
		Paths: []fib_types.FibPath{
			{
				SwIfIndex: uint32(via),
				TableID:   0,
				RpfID:     0,
				Weight:    1,
				Type:      fib_types.FIB_API_PATH_TYPE_NORMAL,
				Flags:     fib_types.FIB_API_PATH_FLAG_NONE,
				Proto:     types.IsV6toFibProto(dst.IP.To4() == nil),
			},
		},
	}
	if nh != nil {
		route.Paths[0].Nh.Address = types.ToVppAddress(nh.IP).Un
	}
	return route
}
