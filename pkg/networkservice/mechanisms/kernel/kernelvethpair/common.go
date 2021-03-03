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

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ethtool"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"
	"github.com/thanhpk/randstr"
	"github.com/vishvananda/netlink"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/mechutils"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/peer"
)

func create(ctx context.Context, conn *networkservice.Connection, isClient bool) error {
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil {
		// TODO - short circuit if already done
		alias := mechutils.ToAlias(conn, isClient)
		if _, err := netlink.LinkByName(linuxIfaceName(alias)); err == nil {
			return nil
		}

		la := netlink.NewLinkAttrs()
		la.Name = randstr.Hex(7)

		// Create the veth pair
		now := time.Now()
		veth := &netlink.Veth{
			LinkAttrs: la,
			PeerName:  linuxIfaceName(alias),
		}
		var l netlink.Link = veth
		if addErr := netlink.LinkAdd(l); addErr != nil {
			return addErr
		}
		log.FromContext(ctx).
			WithField("link.Name", l.Attrs().Name).
			WithField("link.PeerName", veth.PeerName).
			WithField("duration", time.Since(now)).
			WithField("netlink", "LinkAdd").Debug("completed")

		err := ethtool.DisableVethChkSumOffload(veth)
		if err != nil {
			return errors.WithStack(err)
		}

		// Construct the nsHandle and netlink handle for the target namespace for this kernel interface
		nsHandle, err := mechutils.ToNSHandle(mechanism)
		if err != nil {
			return errors.WithStack(err)
		}
		handle, err := mechutils.ToNetlinkHandle(mechanism)
		if err != nil {
			return errors.WithStack(err)
		}
		defer handle.Delete()

		// Set the link l to the correct netns
		now = time.Now()
		if err = netlink.LinkSetNsFd(l, int(nsHandle)); err != nil {
			return errors.Wrapf(err, "unable to change to netns")
		}
		log.FromContext(ctx).
			WithField("link.Name", l.Attrs().Name).
			WithField("duration", time.Since(now)).
			WithField("netlink", "LinkSetNsFd").Debug("completed")

		// Get the link l in the new namespace
		now = time.Now()
		name := l.Attrs().Name
		l, err = handle.LinkByName(name)
		if err != nil {
			log.FromContext(ctx).
				WithField("duration", time.Since(now)).
				WithField("link.Name", name).
				WithField("err", err).
				WithField("netlink", "LinkByName").Debug("error")
			return errors.WithStack(err)
		}
		log.FromContext(ctx).
			WithField("duration", time.Since(now)).
			WithField("link.Name", name).
			WithField("netlink", "LinkByName").Debug("completed")

		name = mechutils.ToInterfaceName(conn, isClient)
		// Set the LinkName
		now = time.Now()
		if err = handle.LinkSetName(l, name); err != nil {
			log.FromContext(ctx).
				WithField("link.Name", l.Attrs().Name).
				WithField("link.NewName", name).
				WithField("duration", time.Since(now)).
				WithField("err", err).
				WithField("netlink", "LinkSetName").Debug("error")
			return errors.WithStack(err)
		}
		log.FromContext(ctx).
			WithField("link.Name", l.Attrs().Name).
			WithField("link.NewName", name).
			WithField("duration", time.Since(now)).
			WithField("netlink", "LinkSetName").Debug("completed")

		// Set the Link Alias
		now = time.Now()
		if err = handle.LinkSetAlias(l, alias); err != nil {
			return errors.WithStack(err)
		}
		log.FromContext(ctx).
			WithField("link.Name", l.Attrs().Name).
			WithField("alias", alias).
			WithField("duration", time.Since(now)).
			WithField("netlink", "LinkSetAlias").Debug("completed")

		// Up the link
		now = time.Now()
		err = handle.LinkSetUp(l)
		if err != nil {
			return errors.WithStack(err)
		}
		log.FromContext(ctx).
			WithField("link.Name", l.Attrs().Name).
			WithField("duration", time.Since(now)).
			WithField("netlink", "LinkSetUp").Debug("completed")

		// Get the peerLink
		now = time.Now()
		peerLink, err := netlink.LinkByName(veth.PeerName)
		if err != nil {
			_ = netlink.LinkDel(l)
			return err
		}
		log.FromContext(ctx).
			WithField("link.Name", veth.PeerName).
			WithField("duration", time.Since(now)).
			WithField("netlink", "LinkByName").Debug("completed")

		// Set Alias of peerLink
		now = time.Now()
		if err = netlink.LinkSetAlias(peerLink, fmt.Sprintf("veth-%s", alias)); err != nil {
			_ = netlink.LinkDel(l)
			_ = netlink.LinkDel(peerLink)
			return err
		}
		log.FromContext(ctx).
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
		log.FromContext(ctx).
			WithField("link.Name", peerLink.Attrs().Name).
			WithField("duration", time.Since(now)).
			WithField("netlink", "LinkSetUp").Debug("completed")

		// Store the link and peerLink
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
			log.FromContext(ctx).
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
