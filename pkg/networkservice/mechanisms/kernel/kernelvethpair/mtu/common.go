// Copyright (c) 2021-2022 Cisco and/or its affiliates.
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

//go:build linux
// +build linux

package mtu

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/tools/peer"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
)

func setMTU(ctx context.Context, conn *networkservice.Connection, isClient bool) error {
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil && mechanism.GetVLAN() == 0 {
		mtu := conn.GetContext().GetMTU()
		if mtu == 0 {
			return nil
		}
		if _, ok := ifindex.Load(ctx, isClient); ok {
			return nil
		}
		peerLink, ok := peer.Load(ctx, isClient)
		if !ok {
			return errors.New("peer link not found")
		}
		now := time.Now()

		if err := netlink.LinkSetMTU(peerLink, int(mtu)); err != nil {
			return errors.Wrapf(err, "error attempting to set MTU on link %q to value %q", peerLink.Attrs().Name, mtu)
		}
		log.FromContext(ctx).
			WithField("link.Name", peerLink.Attrs().Name).
			WithField("MTU", mtu).
			WithField("duration", time.Since(now)).
			WithField("netlink", "LinkSetMTU").Debug("completed")
	}
	return nil
}
