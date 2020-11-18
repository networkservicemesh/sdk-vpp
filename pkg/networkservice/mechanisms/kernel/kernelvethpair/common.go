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
	"net/url"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/trace"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/link"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/netlinkhandle"
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
		la.Name = linuxIfaceName(mechanism.GetInterfaceName(namingConn))
		la.Alias = fmt.Sprintf("server-%s", namingConn.GetId())
		if isClient {
			la.Alias = fmt.Sprintf("client-%s", namingConn.GetId())
		}
		nsHandle, err := toNsHandle(mechanism)
		if err != nil {
			return errors.WithStack(err)
		}
		la.Namespace = netlink.NsFd(nsHandle)
		handle, err := toNetlinkHandle(ctx, nsHandle)
		if err != nil {
			return errors.WithStack(err)
		}
		netlinkhandle.Store(ctx, isClient, handle)
		l := &netlink.Veth{
			LinkAttrs: la,
			PeerName:  linuxIfaceName(la.Alias),
		}
		now := time.Now()
		if addErr := netlink.LinkAdd(l); addErr != nil {
			return addErr
		}
		trace.Log(ctx).
			WithField("link.Name", l.Name).
			WithField("link.PeerName", l.PeerName).
			WithField("link.Alias", l.Alias).
			WithField("link.OperState", l.OperState).
			WithField("duration", time.Since(now)).
			WithField("netlink", "LinkAdd").Debug("completed")

		now = time.Now()
		peerLink, err := netlink.LinkByName(linuxIfaceName(la.Alias))
		if err != nil {
			_ = netlink.LinkDel(l)
			return err
		}
		trace.Log(ctx).
			WithField("link.Name", linuxIfaceName(la.Alias)).
			WithField("duration", time.Since(now)).
			WithField("netlink", "LinkByName").Debug("completed")

		now = time.Now()
		if err = netlink.LinkSetAlias(peerLink, fmt.Sprintf("veth-%s", la.Alias)); err != nil {
			_ = netlink.LinkDel(l)
			_ = netlink.LinkDel(peerLink)
			return err
		}
		trace.Log(ctx).
			WithField("link.Name", peerLink.Attrs().Name).
			WithField("alias", fmt.Sprintf("veth-%s", la.Alias)).
			WithField("duration", time.Since(now)).
			WithField("netlink", "LinkSetAlias").Debug("completed")

		now = time.Now()
		err = handle.LinkSetUp(l)
		if err != nil {
			_ = netlink.LinkDel(l)
			_ = netlink.LinkDel(peerLink)
			return err
		}
		trace.Log(ctx).
			WithField("link.Name", l.Attrs().Name).
			WithField("duration", time.Since(now)).
			WithField("netlink", "LinkSetUp").Debug("completed")

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

		link.Store(ctx, isClient, l)
		peer.Store(ctx, isClient, peerLink)
	}
	return nil
}

func del(ctx context.Context, conn *networkservice.Connection, isClient bool) error {
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil {
		if peerLink, ok := peer.Load(ctx, isClient); ok {
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

func toNsHandle(mechanism *kernel.Mechanism) (netns.NsHandle, error) {
	u, err := url.Parse(mechanism.GetNetNSURL())
	if err != nil {
		return 0, err
	}
	if u.Scheme != "file" {
		return 0, errors.Errorf("NetNSURL Scheme required to be %q actual %q", "file", u.Scheme)
	}
	return netns.GetFromPath(u.Path)
}

func toNetlinkHandle(ctx context.Context, nsHandle netns.NsHandle) (*netlink.Handle, error) {
	curNsHandle, err := netns.Get()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	now := time.Now()
	handle, err := netlink.NewHandleAtFrom(nsHandle, curNsHandle)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	trace.Log(ctx).
		WithField("duration", time.Since(now)).
		WithField("netlink", "NewHandleAtFrom").Debug("completed")
	return handle, nil
}
