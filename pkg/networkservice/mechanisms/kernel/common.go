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

package kernel

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
)

func create(ctx context.Context, conn *networkservice.Connection, isClient bool) error {
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil {
		// If we already have a netlink handle for this connection, we've already done the work here and don't
		// need to repeat it.
		_, ok := netlinkhandle.Load(ctx, isClient)
		if ok {
			return nil
		}

		// Get the link l - this will have been created by kernelvethpair or kerneltap
		l, ok := link.Load(ctx, isClient)
		if !ok {
			return errors.New("unable to find link")
		}

		// Construct the nsHandle and netlink handle for the target namespace for this kernel interface
		nsHandle, err := toNSHandle(mechanism)
		if err != nil {
			return errors.WithStack(err)
		}
		handle, err := toNetlinkHandle(ctx, nsHandle)
		if err != nil {
			return errors.WithStack(err)
		}
		netlinkhandle.Store(ctx, isClient, handle)

		// Set the link l to the correct netns
		now := time.Now()
		if err = netlink.LinkSetNsFd(l, int(nsHandle)); err != nil {
			return errors.Wrapf(err, "unable to change to netns")
		}
		trace.Log(ctx).
			WithField("link.Name", l.Attrs().Name).
			WithField("duration", time.Since(now)).
			WithField("netlink", "LinkSetNsFd").Debug("completed")

		// Naming is tricky.  We want to name based on either the next or prev connection id depending on whether we
		// are on the client or server side.  Since this chain element is designed for use in a Forwarder,
		// if we are on the client side, we want to name based on the connection id from the NSE that is Next
		// if we are not the client, we want to name for the connection of of the client addressing us, which is Prev
		namingConn := conn.Clone()
		namingConn.Id = namingConn.GetPrevPathSegment().GetId()
		if isClient {
			namingConn.Id = namingConn.GetNextPathSegment().GetId()
		}
		name := mechanism.GetInterfaceName(namingConn)
		alias := fmt.Sprintf("server-%s", namingConn.GetId())
		if isClient {
			alias = fmt.Sprintf("client-%s", namingConn.GetId())
		}

		// Set the LinkName
		now = time.Now()
		if err = handle.LinkSetName(l, name); err != nil {
			return errors.WithStack(err)
		}
		trace.Log(ctx).
			WithField("link.Name", l.Attrs().Name).
			WithField("link.NewName", name).
			WithField("duration", time.Since(now)).
			WithField("netlink", "LinkSetName").Debug("completed")

		// Set the Link Alias
		now = time.Now()
		if err = handle.LinkSetAlias(l, alias); err != nil {
			return errors.WithStack(err)
		}
		trace.Log(ctx).
			WithField("link.Name", name).
			WithField("alias", alias).
			WithField("duration", time.Since(now)).
			WithField("netlink", "LinkSetAlias").Debug("completed")

		// Up the link
		now = time.Now()
		err = handle.LinkSetUp(l)
		if err != nil {
			return errors.WithStack(err)
		}
		trace.Log(ctx).
			WithField("link.Name", l.Attrs().Name).
			WithField("duration", time.Since(now)).
			WithField("netlink", "LinkSetUp").Debug("completed")
	}
	return nil
}

func del(ctx context.Context, conn *networkservice.Connection, isClient bool) error {
	// We do nothing here, because the kernelvethpair or kerneltap chain elements handle deleting
	return nil
}

func toNSHandle(mechanism *kernel.Mechanism) (netns.NsHandle, error) {
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
	curNSHandle, err := netns.Get()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	now := time.Now()
	handle, err := netlink.NewHandleAtFrom(nsHandle, curNSHandle)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	trace.Log(ctx).
		WithField("duration", time.Since(now)).
		WithField("netlink", "NewHandleAtFrom").Debug("completed")
	return handle, nil
}
