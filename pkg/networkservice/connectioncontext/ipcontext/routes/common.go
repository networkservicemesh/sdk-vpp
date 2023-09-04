// Copyright (c) 2020-2023 Cisco and/or its affiliates.
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
	"time"

	"github.com/pkg/errors"
	"go.fd.io/govpp/api"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/govpp/binapi/fib_types"
	"github.com/networkservicemesh/govpp/binapi/interface_types"
	"github.com/networkservicemesh/govpp/binapi/ip"
	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/vrf"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/types"
)

func addDel(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection, isClient, isAdd bool) error {
	swIfIndex, ok := ifindex.Load(ctx, isClient)
	if !ok {
		return nil
	}
	var routes []*networkservice.Route
	if isClient {
		// Prepend any routes needed to be able to reach the SrcIPs
		routes = conn.GetContext().GetIpContext().GetDstIPRoutes()
		routes = append(routes, conn.GetContext().GetIpContext().GetSrcRoutesWithExplicitNextHop()...)
	} else {
		routes = conn.GetContext().GetIpContext().GetSrcIPRoutes()
		routes = append(routes, conn.GetContext().GetIpContext().GetDstRoutesWithExplicitNextHop()...)
	}

	for _, route := range routes {
		if err := routeAddDel(ctx, vppConn, swIfIndex, isClient, isAdd, route); err != nil {
			return err
		}
	}

	return nil
}

func routeAddDel(ctx context.Context, vppConn api.Connection, swIfIndex interface_types.InterfaceIndex, isClient, isAdd bool, route *networkservice.Route) error {
	if route.GetPrefixIPNet() == nil {
		return errors.New("vppRoute prefix must not be nil")
	}
	isIPV6 := route.GetPrefixIPNet().IP.To4() == nil
	tableID, _ := vrf.Load(ctx, isClient, isIPV6)
	vppRoute := toRoute(route, swIfIndex, tableID)
	now := time.Now()
	if _, err := ip.NewServiceClient(vppConn).IPRouteAddDel(ctx, &ip.IPRouteAddDel{
		IsAdd:       isAdd,
		IsMultipath: false,
		Route:       vppRoute,
	}); err != nil {
		return errors.Wrap(err, "vppapi IPRouteAddDel returned error")
	}
	log.FromContext(ctx).
		WithField("swIfIndex", swIfIndex).
		WithField("prefix", vppRoute.Prefix).
		WithField("isIpV6", isIPV6).
		WithField("tableID", tableID).
		WithField("isAdd", isAdd).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "IPRouteAddDel").Info("completed")
	return nil
}

func toRoute(route *networkservice.Route, via interface_types.InterfaceIndex, tableID uint32) ip.IPRoute {
	prefix := route.GetPrefixIPNet()
	rv := ip.IPRoute{
		TableID:    tableID,
		StatsIndex: 0,
		Prefix:     types.ToVppPrefix(prefix),
		NPaths:     1,
		Paths: []fib_types.FibPath{
			{
				SwIfIndex: uint32(via),
				TableID:   0,
				RpfID:     0,
				Weight:    1,
				Type:      fib_types.FIB_API_PATH_TYPE_NORMAL,
				Flags:     fib_types.FIB_API_PATH_FLAG_NONE,
				Proto:     types.IsV6toFibProto(prefix.IP.To4() == nil),
			},
		},
	}
	nh := route.GetNextHopIP()
	if nh != nil {
		rv.Paths[0].Nh.Address = types.ToVppAddress(nh).Un
	}
	return rv
}
