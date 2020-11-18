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

// +build linux

package routes

import (
	"context"
	"net"
	"time"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/trace"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/link"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/netlinkhandle"
)

func create(ctx context.Context, conn *networkservice.Connection, isClient bool) error {
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil {
		l, ok := link.Load(ctx, isClient)
		if !ok {
			return nil
		}
		handle, ok := netlinkhandle.Load(ctx, isClient)
		if !ok {
			return errors.Errorf("did not find netlink handle with which to program routes")
		}
		from := conn.GetContext().GetIpContext().GetSrcIPNet()
		to := conn.GetContext().GetIpContext().GetDstIPNet()
		if isClient {
			from = conn.GetContext().GetIpContext().GetDstIPNet()
			to = conn.GetContext().GetIpContext().GetSrcIPNet()
		}
		routes := conn.GetContext().GetIpContext().GetDstRoutes()
		if isClient {
			routes = conn.GetContext().GetIpContext().GetSrcRoutes()
		}
		for _, route := range routes {
			if err := routeAdd(ctx, handle, l, netlink.SCOPE_UNIVERSE, route.GetPrefixIPNet(), to); err != nil {
				return err
			}
		}
		if to != nil && !to.Contains(from.IP) {
			if err := routeAdd(ctx, handle, l, netlink.SCOPE_LINK, to, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

func routeAdd(ctx context.Context, handle *netlink.Handle, l netlink.Link, scope netlink.Scope, prefix, gw *net.IPNet) error {
	route := &netlink.Route{
		LinkIndex: l.Attrs().Index,
		Scope:     scope,
		Dst:       prefix,
	}
	if gw != nil {
		route.Gw = gw.IP
	}
	now := time.Now()
	if err := handle.RouteAdd(route); err != nil {
		return errors.WithStack(err)
	}
	trace.Log(ctx).
		WithField("link.Name", l.Attrs().Name).
		WithField("route.Dst", route.Dst).
		WithField("route.Gw", route.Gw).
		WithField("route.Scope", route.Scope).
		WithField("duration", time.Since(now)).
		WithField("netlink", "RouteAdd").Debug("completed")
	return nil
}
