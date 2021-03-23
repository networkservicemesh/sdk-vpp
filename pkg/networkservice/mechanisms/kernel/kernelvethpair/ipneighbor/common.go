// Copyright (c) 2021 Cisco and/or its affiliates.
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

package ipneighbor

import (
	"context"
	"fmt"
	"net"
	"time"

	"git.fd.io/govpp.git/api"
	"github.com/edwarnicke/govpp/binapi/interface_types"
	"github.com/edwarnicke/govpp/binapi/ip_neighbor"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/link"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/mechutils"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/peer"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/types"
)

func addDel(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection, isClient, isAdd bool) error {
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil {
		srcNet := conn.GetContext().GetIpContext().GetSrcIPNet()
		dstNet := conn.GetContext().GetIpContext().GetDstIPNet()
		if isClient {
			srcNet = conn.GetContext().GetIpContext().GetDstIPNet()
			dstNet = conn.GetContext().GetIpContext().GetSrcIPNet()
		}
		swIfIndex, ok := ifindex.Load(ctx, isClient)
		if !ok {
			return nil
		}
		l, ok := link.Load(ctx, isClient)
		if !ok {
			return nil
		}
		if l == nil || l.Attrs() == nil || l.Attrs().HardwareAddr == nil {
			panic(fmt.Sprintf("unable to construct ip neighborL %+v", l))
		}
		if srcNet != nil {
			if err := addDelVPP(ctx, vppConn, isAdd, swIfIndex, srcNet, l); err != nil {
				return err
			}
		}
		peerLink, ok := peer.Load(ctx, isClient)
		if !ok {
			return nil
		}
		if peerLink == nil || peerLink.Attrs() == nil || peerLink.Attrs().HardwareAddr == nil {
			panic(fmt.Sprintf("unable to construct peer ip neighborL %+v", peerLink))
		}
		if dstNet != nil {
			if err := addDelKernel(ctx, isAdd, mechanism, l, peerLink, dstNet); err != nil {
				return err
			}
		}
	}
	return nil
}

func addDelVPP(ctx context.Context, vppConn api.Connection, isAdd bool, swIfIndex interface_types.InterfaceIndex, srcNet *net.IPNet, l netlink.Link) error {
	now := time.Now()
	ipNeighborAddDel := &ip_neighbor.IPNeighborAddDel{
		IsAdd: isAdd,
		Neighbor: ip_neighbor.IPNeighbor{
			SwIfIndex:  swIfIndex,
			Flags:      ip_neighbor.IP_API_NEIGHBOR_FLAG_STATIC,
			MacAddress: types.ToVppMacAddress(&l.Attrs().HardwareAddr),
			IPAddress:  types.ToVppAddress(srcNet.IP),
		},
	}
	_, err := ip_neighbor.NewServiceClient(vppConn).IPNeighborAddDel(ctx, ipNeighborAddDel)
	if err != nil {
		return errors.WithStack(err)
	}
	log.FromContext(ctx).
		WithField("swIfIndex", ipNeighborAddDel.Neighbor.SwIfIndex).
		WithField("flags", ipNeighborAddDel.Neighbor.Flags).
		WithField("macaddress", ipNeighborAddDel.Neighbor.MacAddress).
		WithField("ipaddress", ipNeighborAddDel.Neighbor.IPAddress).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "IPNeighborAddDel").Debug("completed")
	return nil
}

func addDelKernel(ctx context.Context, isAdd bool, mechanism *kernel.Mechanism, l, peerLink netlink.Link, dstNet *net.IPNet) error {
	now := time.Now()
	handle, err := mechutils.ToNetlinkHandle(mechanism)
	if err != nil {
		return errors.WithStack(err)
	}
	defer handle.Delete()
	neigh := &netlink.Neigh{
		LinkIndex:    l.Attrs().Index,
		IP:           dstNet.IP,
		State:        netlink.NUD_PERMANENT,
		HardwareAddr: peerLink.Attrs().HardwareAddr,
	}
	if isAdd {
		if err = handle.NeighAdd(neigh); err != nil {
			return errors.WithStack(err)
		}
		log.FromContext(ctx).
			WithField("linkIndex", neigh.LinkIndex).
			WithField("ip", neigh.IP).
			WithField("state", neigh.State).
			WithField("hardwareAddr", neigh.HardwareAddr).
			WithField("duration", time.Since(now)).
			WithField("netlink", "NeighAdd").Debug("completed")
		return nil
	}
	if err = handle.NeighDel(neigh); err != nil {
		return errors.WithStack(err)
	}
	log.FromContext(ctx).
		WithField("linkIndex", neigh.LinkIndex).
		WithField("ip", neigh.IP).
		WithField("state", neigh.State).
		WithField("hardwareAddr", neigh.HardwareAddr).
		WithField("duration", time.Since(now)).
		WithField("netlink", "NeighDel").Debug("completed")
	return nil
}
