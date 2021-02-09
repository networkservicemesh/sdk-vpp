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

package afpacket

import (
	"context"
	"time"

	"git.fd.io/govpp.git/api"
	"github.com/edwarnicke/govpp/binapi/af_packet"
	interfaces "github.com/edwarnicke/govpp/binapi/interface"
	"github.com/edwarnicke/govpp/binapi/interface_types"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/up"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/peer"
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
		rsp, err := af_packet.NewServiceClient(vppConn).AfPacketCreate(ctx, &af_packet.AfPacketCreate{
			HostIfName: peerLink.Attrs().Name,
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

func del(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection, isClient bool) error {
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil {
		swIfIndex, ok := ifindex.Load(ctx, isClient)
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
