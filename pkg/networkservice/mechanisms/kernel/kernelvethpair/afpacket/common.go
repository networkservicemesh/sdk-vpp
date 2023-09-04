// Copyright (c) 2020-2023 Cisco and/or its affiliates.
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

package afpacket

import (
	"context"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/govpp/binapi/af_packet"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"
	"go.fd.io/govpp/api"

	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/tools/peer"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/up"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/types"
)

func create(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection, isClient bool) error {
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil {
		if _, ok := ifindex.Load(ctx, isClient); ok {
			return nil
		}
		peerLink, ok := peer.Load(ctx, isClient)
		if !ok {
			return errors.New("peer link not found")
		}
		now := time.Now()
		rsp, err := af_packet.NewServiceClient(vppConn).AfPacketCreateV3(ctx, &af_packet.AfPacketCreateV3{
			Mode:        af_packet.AF_PACKET_API_MODE_ETHERNET,
			HostIfName:  peerLink.Attrs().Name,
			HwAddr:      types.ToVppMacAddress(&peerLink.Attrs().HardwareAddr),
			RxFrameSize: 10240,
			TxFrameSize: 10240,
			Flags:       af_packet.AF_PACKET_API_FLAG_VERSION_2,
		})
		if err != nil {
			return errors.Wrap(err, "vppapi AfPacketCreateV3 returned error")
		}
		log.FromContext(ctx).
			WithField("swIfIndex", rsp.SwIfIndex).
			WithField("duration", time.Since(now)).
			WithField("vppapi", "AfPacketCreateV3").Debug("completed")
		ifindex.Store(ctx, isClient, rsp.SwIfIndex)

		up.Store(ctx, isClient, true)
	}
	return nil
}

func del(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection, isClient bool) error {
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil {
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
			return errors.Wrap(err, "vppapi AfPacketDelete returned error")
		}
		log.FromContext(ctx).
			WithField("swIfIndex", swIfIndex).
			WithField("duration", time.Since(now)).
			WithField("vppapi", "AfPacketDelete").Debug("completed")
		return nil
	}
	return nil
}
