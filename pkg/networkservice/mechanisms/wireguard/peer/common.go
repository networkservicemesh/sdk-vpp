// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

package peer

import (
	"context"
	"time"

	"git.fd.io/govpp.git/api"
	"github.com/pkg/errors"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/edwarnicke/govpp/binapi/ip_types"
	"github.com/edwarnicke/govpp/binapi/wireguard"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	wireguardMech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/wireguard"
	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/types"
)

func getKey(mech *wireguardMech.Mechanism, isClient bool) string {
	if isClient {
		return mech.DstPublicKey()
	}
	return mech.SrcPublicKey()
}

func createPeer(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection, isClient bool) error {
	if mechanism := wireguardMech.ToMechanism(conn.GetMechanism()); mechanism != nil {
		pubKeyStr := getKey(mechanism, isClient)
		_, ok := Load(ctx, isClient, pubKeyStr)
		if ok {
			return nil
		}
		ifIdx, ok := ifindex.Load(ctx, isClient)
		if !ok {
			return nil
		}

		now := time.Now()
		pubKeyBin, e := wgtypes.ParseKey(pubKeyStr)
		if e != nil {
			return errors.WithStack(e)
		}
		peer := wireguard.WireguardPeer{
			SwIfIndex:           ifIdx,
			PublicKey:           pubKeyBin[:],
			PersistentKeepalive: 10,
		}
		if !isClient {
			for _, ip := range conn.GetContext().GetIpContext().GetSrcIpAddrs() {
				allowedIP, e := ip_types.ParsePrefix(ip)
				if e != nil {
					return errors.WithStack(e)
				}
				peer.AllowedIps = append(peer.AllowedIps, allowedIP)
			}
			peer.NAllowedIps = uint8(len(conn.GetContext().GetIpContext().GetSrcIpAddrs()))
			peer.Port = mechanism.SrcPort()
			peer.Endpoint = types.ToVppAddress(mechanism.SrcIP())
		} else {
			for _, ip := range conn.GetContext().GetIpContext().GetDstIpAddrs() {
				allowedIP, e := ip_types.ParsePrefix(ip)
				if e != nil {
					return errors.WithStack(e)
				}
				peer.AllowedIps = append(peer.AllowedIps, allowedIP)
			}
			peer.NAllowedIps = uint8(len(conn.GetContext().GetIpContext().GetDstIpAddrs()))
			peer.Port = mechanism.DstPort()
			peer.Endpoint = types.ToVppAddress(mechanism.DstIP())
		}

		wgPeerCreate := &wireguard.WireguardPeerAdd{
			Peer: peer,
		}
		rspPeer, err := wireguard.NewServiceClient(vppConn).WireguardPeerAdd(ctx, wgPeerCreate)
		if err != nil {
			return errors.WithStack(err)
		}
		log.FromContext(ctx).
			WithField("PeerIndex", rspPeer.PeerIndex).
			WithField("duration", time.Since(now)).
			WithField("vppapi", "WireguardPeerAdd").Debug("completed")
		Store(ctx, isClient, pubKeyStr, rspPeer.PeerIndex)
	}
	return nil
}

func delPeer(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection, isClient bool) error {
	if mechanism := wireguardMech.ToMechanism(conn.GetMechanism()); mechanism != nil {
		peerIdx, ok := LoadAndDelete(ctx, isClient, getKey(mechanism, isClient))
		if !ok {
			return nil
		}
		now := time.Now()
		wgPeerRem := &wireguard.WireguardPeerRemove{
			PeerIndex: peerIdx,
		}
		_, err := wireguard.NewServiceClient(vppConn).WireguardPeerRemove(ctx, wgPeerRem)
		if err != nil {
			return errors.WithStack(err)
		}
		log.FromContext(ctx).
			WithField("duration", time.Since(now)).
			WithField("vppapi", "WireguardPeerRemove").Debug("completed")
	}
	return nil
}
