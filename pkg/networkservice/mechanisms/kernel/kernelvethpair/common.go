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

package kernelvethpair

import (
	"context"
	"fmt"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/trace"
	"github.com/thanhpk/randstr"
	"github.com/vishvananda/netlink"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/link"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/peer"
)

func create(ctx context.Context, conn *networkservice.Connection, isClient bool) error {
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil {
		// TODO - short circuit if already done
		la := netlink.NewLinkAttrs()

		namingConn := conn.Clone()
		namingConn.Id = namingConn.GetPrevPathSegment().GetId()
		if isClient {
			namingConn.Id = namingConn.GetNextPathSegment().GetId()
		}
		la.Name = randstr.Hex(7)
		alias := fmt.Sprintf("server-%s", namingConn.GetId())
		if isClient {
			alias = fmt.Sprintf("client-%s", namingConn.GetId())
		}

		// Create the veth pair
		now := time.Now()
		l := &netlink.Veth{
			LinkAttrs: la,
			PeerName:  linuxIfaceName(alias),
		}
		if addErr := netlink.LinkAdd(l); addErr != nil {
			return addErr
		}
		trace.Log(ctx).
			WithField("link.Name", l.Name).
			WithField("link.PeerName", l.PeerName).
			WithField("duration", time.Since(now)).
			WithField("netlink", "LinkAdd").Debug("completed")

		// Get the peerLink
		now = time.Now()
		peerLink, err := netlink.LinkByName(l.PeerName)
		if err != nil {
			_ = netlink.LinkDel(l)
			return err
		}
		trace.Log(ctx).
			WithField("link.Name", l.PeerName).
			WithField("duration", time.Since(now)).
			WithField("netlink", "LinkByName").Debug("completed")

		// Set Alias of peerLink
		now = time.Now()
		if err = netlink.LinkSetAlias(peerLink, fmt.Sprintf("veth-%s", alias)); err != nil {
			_ = netlink.LinkDel(l)
			_ = netlink.LinkDel(peerLink)
			return err
		}
		trace.Log(ctx).
			WithField("link.Name", peerLink.Attrs().Name).
			WithField("peerLink", fmt.Sprintf("veth-%s", alias)).
			WithField("duration", time.Since(now)).
			WithField("netlink", "LinkSetAlias").Debug("completed")

		// Up the peerLink
		now = time.Now()
		err = netlink.LinkSetUp(peerLink)
		if err != nil {
			_ = netlink.LinkDel(l)
			_ = netlink.LinkDel(peerLink)
			return err
		}
		trace.Log(ctx).
			WithField("link.Name", peerLink.Attrs().Name).
			WithField("duration", time.Since(now)).
			WithField("netlink", "LinkSetUp").Debug("completed")

		// Store the link and peerLink
		link.Store(ctx, isClient, l)
		peer.Store(ctx, isClient, peerLink)
	}
	return nil
}

func del(ctx context.Context, conn *networkservice.Connection, isClient bool) error {
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil {
		if peerLink, ok := peer.Load(ctx, isClient); ok {
			// Delete the peerLink which deletes all associated pair partners, routes, etc
			now := time.Now()
			if err := netlink.LinkDel(peerLink); err != nil {
				return err
			}
			trace.Log(ctx).
				WithField("link.Name", peerLink.Attrs().Name).
				WithField("duration", time.Since(now)).
				WithField("netlink", "LinkDel").Debug("completed")
		}
	}
	return nil
}

func linuxIfaceName(ifaceName string) string {
	if len(ifaceName) <= kernel.LinuxIfMaxLength {
		return ifaceName
	}
	return ifaceName[:kernel.LinuxIfMaxLength]
}
