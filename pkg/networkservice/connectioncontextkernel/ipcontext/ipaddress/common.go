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

package ipaddress

import (
	"context"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/mechutils"
)

func create(ctx context.Context, conn *networkservice.Connection, isClient bool) error {
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil {
		// Note: These are switched from normal because if we are the client, we need to assign the IP
		// in the Endpoints NetNS for the Dst.  If we are the *server* we need to assign the IP for the
		// clients NetNS (ie the source).
		ipNets := conn.GetContext().GetIpContext().GetSrcIPNets()
		if isClient {
			ipNets = conn.GetContext().GetIpContext().GetDstIPNets()
		}
		if ipNets == nil {
			return nil
		}

		handle, err := mechutils.ToNetlinkHandle(mechanism)
		if err != nil {
			return errors.WithStack(err)
		}
		defer handle.Delete()

		l, err := handle.LinkByName(mechutils.ToInterfaceName(conn, isClient))
		if err != nil {
			return errors.WithStack(err)
		}

		for _, ipNet := range ipNets {
			now := time.Now()
			addr := &netlink.Addr{
				IPNet: ipNet,
			}
			// Turns out IPv6 uses Duplicate Address Detection (DAD) which
			// we don't need here and which can cause it to take more than a second
			// before anything *works* (even though the interface is up).  This causes
			// cryptic error messages.  To avoid, we use the flag to disable DAD for
			// any IPv6 addresses.
			if ipNet != nil && ipNet.IP.To4() == nil {
				addr.Flags |= unix.IFA_F_NODAD
			}
			if err := handle.AddrReplace(l, addr); err != nil {
				return err
			}
			log.FromContext(ctx).
				WithField("link.Name", l.Attrs().Name).
				WithField("Addr", ipNet.String()).
				WithField("duration", time.Since(now)).
				WithField("netlink", "AddrAdd").Debug("completed")
		}
	}
	return nil
}
