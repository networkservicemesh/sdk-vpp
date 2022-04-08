// Copyright (c) 2020-2022 Cisco and/or its affiliates.
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

package afpacket

import (
	"context"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/dumptool"
	"github.com/vishvananda/netlink"
	"io"
	"time"

	"git.fd.io/govpp.git/api"
	"github.com/edwarnicke/govpp/binapi/af_packet"
	interfaces "github.com/edwarnicke/govpp/binapi/interface"
	"github.com/edwarnicke/govpp/binapi/interface_types"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/tools/peer"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/up"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/types"
)

func create(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection, dumpMap *dumptool.Map, isClient bool) error {
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil {
		if val, loaded := dumpMap.LoadAndDelete(conn.GetId()); loaded {
			ifindex.Store(ctx, isClient, val.(interface_types.InterfaceIndex))
		}

		if _, ok := ifindex.Load(ctx, isClient); ok {
			return nil
		}
		peerLink, ok := peer.Load(ctx, isClient)
		if !ok {
			return errors.New("peer link not found")
		}
		now := time.Now()
		rsp, err := af_packet.NewServiceClient(vppConn).AfPacketCreate(ctx, &af_packet.AfPacketCreate{
			HostIfName: peerLink.Attrs().Name,
			HwAddr:     types.ToVppMacAddress(&peerLink.Attrs().HardwareAddr),
		})
		if err != nil {
			return errors.WithStack(err)
		}
		log.FromContext(ctx).
			WithField("swIfIndex", rsp.SwIfIndex).
			WithField("duration", time.Since(now)).
			WithField("vppapi", "AfPacketCreate").Debug("completed")
		ifindex.Store(ctx, isClient, rsp.SwIfIndex)

		now = time.Now()
		if _, err := interfaces.NewServiceClient(vppConn).SwInterfaceSetRxMode(ctx, &interfaces.SwInterfaceSetRxMode{
			SwIfIndex: rsp.SwIfIndex,
			Mode:      interface_types.RX_MODE_API_ADAPTIVE,
		}); err != nil {
			return errors.WithStack(err)
		}
		log.FromContext(ctx).
			WithField("swIfIndex", rsp.SwIfIndex).
			WithField("mode", interface_types.RX_MODE_API_ADAPTIVE).
			WithField("duration", time.Since(now)).
			WithField("vppapi", "SwInterfaceSetRxMode").Debug("completed")
		up.Store(ctx, isClient, true)
	}
	return nil
}

func del(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection, dumpMap *dumptool.Map, isClient bool) error {
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil {
		if val, loaded := dumpMap.LoadAndDelete(conn.GetId()); loaded {
			ifindex.Store(ctx, isClient, val.(interface_types.InterfaceIndex))
		}

		swIfIndex, ok := ifindex.LoadAndDelete(ctx, isClient)
		if !ok {
			return nil
		}
		peerLink, ok := peer.Load(ctx, isClient)
		if !ok {
			return errors.New("peer link not found")
		}
		now := time.Now()
		_, err := af_packet.NewServiceClient(vppConn).AfPacketDelete(ctx, &af_packet.AfPacketDelete{
			HostIfName: peerLink.Attrs().Name,
		})
		if err != nil {
			return errors.WithStack(err)
		}
		log.FromContext(ctx).
			WithField("swIfIndex", swIfIndex).
			WithField("duration", time.Since(now)).
			WithField("vppapi", "AfPacketDelete").Debug("completed")
		return nil
	}
	return nil
}

func dump(ctx context.Context, vppConn api.Connection, podName string, timeout time.Duration, isClient bool) (*dumptool.Map, error) {
	return dumptool.DumpInterfaces(ctx, vppConn, podName, timeout, isClient,
		/* Function on dump */
		func(details *interfaces.SwInterfaceDetails) (interface{}, error) {
			if details.InterfaceDevType == dumptool.DevTypeAfPacket {
				return details.SwIfIndex, nil
			}
			return nil, errors.New("Doesn't match the af_packet interface")
		},
		/* Function on delete */
		func(val interface{}) error {
			swIfIndex := val.(interface_types.InterfaceIndex)
			afClient, err := af_packet.NewServiceClient(vppConn).AfPacketDump(ctx, &af_packet.AfPacketDump{})
			if err != nil {
				return err
			}
			defer func() { _ = afClient.Close() }()

			for {
				afDetails, err := afClient.Recv()
				if err == io.EOF {
					break
				}
				if afDetails == nil || afDetails.SwIfIndex != swIfIndex {
					continue
				}

				if _, err := af_packet.NewServiceClient(vppConn).AfPacketDelete(ctx, &af_packet.AfPacketDelete{
					HostIfName: afDetails.HostIfName,
				}); err != nil {
					return err
				}
				return nil
			}
			return netlink.LinkDel(val.(netlink.Link))
		})
}
