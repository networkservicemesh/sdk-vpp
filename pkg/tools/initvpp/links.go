// Copyright (c) 2020-2021 Cisco and/or its affiliates.
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

// Package initvpp contains initialization code for vpp
package initvpp

import (
	"context"
	"net"
	"time"

	"git.fd.io/govpp.git/api"
	"github.com/edwarnicke/govpp/binapi/af_packet"
	interfaces "github.com/edwarnicke/govpp/binapi/interface"
	"github.com/edwarnicke/govpp/binapi/interface_types"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"

	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/types"
)

// LinkByIP - get link by it's IP
func LinkByIP(ctx context.Context, ipaddress net.IP) (netlink.Link, error) {
	if ipaddress == nil {
		return defaultRouteLink(ctx)
	}
	links, err := netlink.LinkList()
	if err != nil {
		return nil, errors.Wrap(err, "unable to get links")
	}
	for _, link := range links {
		addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
		if err != nil {
			return nil, errors.Wrap(err, "could not find links for default routes")
		}
		for _, addr := range addrs {
			if addr.IPNet != nil && addr.IPNet.IP.Equal(ipaddress) {
				return link, nil
			}
		}
	}
	return nil, nil
}

func defaultRouteLink(ctx context.Context) (netlink.Link, error) {
	now := time.Now()
	routes, err := netlink.RouteList(nil, netlink.FAMILY_ALL)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get routes")
	}

	log.FromContext(ctx).
		WithField("duration", time.Since(now)).
		WithField("netlink", "RouteList").Debug("completed")

	for _, route := range routes {
		// Is it a default route?
		if route.Dst != nil {
			ones, _ := route.Dst.Mask.Size()
			if ones == 0 && (route.Dst.IP.Equal(net.IPv4zero) || route.Dst.IP.Equal(net.IPv6zero)) {
				return netlink.LinkByIndex(route.LinkIndex)
			}
			continue
		}
		if route.Scope == netlink.SCOPE_UNIVERSE {
			return netlink.LinkByIndex(route.LinkIndex)
		}
	}
	return nil, errors.New("no link found for default route")
}

// CreateAfPacket - creates veth pair of an existing AF_PACKET interface
func CreateAfPacket(ctx context.Context, vppConn api.Connection, link netlink.Link) (interface_types.InterfaceIndex, error) {
	afPacketCreate := &af_packet.AfPacketCreate{
		HwAddr:     types.ToVppMacAddress(&link.Attrs().HardwareAddr),
		HostIfName: link.Attrs().Name,
	}
	now := time.Now()
	afPacketCreateRsp, err := af_packet.NewServiceClient(vppConn).AfPacketCreate(ctx, afPacketCreate)
	if err != nil {
		return 0, err
	}
	log.FromContext(ctx).
		WithField("swIfIndex", afPacketCreateRsp.SwIfIndex).
		WithField("hwaddr", afPacketCreate.HwAddr).
		WithField("hostIfName", afPacketCreate.HostIfName).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "AfPacketCreate").Debug("completed")

	if err := setMtu(ctx, vppConn, link, afPacketCreateRsp.SwIfIndex); err != nil {
		return 0, err
	}
	return afPacketCreateRsp.SwIfIndex, nil
}

func setMtu(ctx context.Context, vppConn api.Connection, link netlink.Link, swIfIndex interface_types.InterfaceIndex) error {
	now := time.Now()
	setMtu := &interfaces.HwInterfaceSetMtu{
		SwIfIndex: swIfIndex,
		Mtu:       uint16(link.Attrs().MTU),
	}
	_, err := interfaces.NewServiceClient(vppConn).HwInterfaceSetMtu(ctx, setMtu)
	if err != nil {
		return err
	}
	log.FromContext(ctx).
		WithField("swIfIndex", setMtu.SwIfIndex).
		WithField("MTU", setMtu.Mtu).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "HwInterfaceSetMtu").Debug("completed")
	return nil
}
