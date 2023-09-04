// Copyright (c) 2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023 Cisco and/or its affiliates.
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

package peerup

import (
	"context"
	"os"
	"time"

	"github.com/networkservicemesh/govpp/binapi/interface_types"
	"github.com/networkservicemesh/govpp/binapi/wireguard"
	"github.com/pkg/errors"
	"go.fd.io/govpp/api"

	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/wireguard/peer"
)

// Connection - simply combines tha api.Connection and api.ChannelProvider interfaces
type Connection interface {
	api.Connection
	api.ChannelProvider
}

func waitForPeerUp(ctx context.Context, vppConn Connection, pubKey string, isClient bool) error {
	peerIndex, ok := peer.Load(ctx, isClient, pubKey)
	if !ok {
		return errors.New("Peer not found")
	}

	apiChannel, err := getAPIChannel(ctx, vppConn, peerIndex)
	if err != nil {
		return err
	}

	notifCh := make(chan api.Message, 256)
	subscription, err := apiChannel.SubscribeNotification(notifCh, &wireguard.WireguardPeerEvent{})
	if err != nil {
		return errors.Wrap(err, "failed to subscribe for receiving of the specified notification messages via provided Go channel")
	}
	defer func() { _ = subscription.Unsubscribe() }()

	now := time.Now()

	dp, err := wireguard.NewServiceClient(vppConn).WireguardPeersDump(ctx, &wireguard.WireguardPeersDump{
		PeerIndex: peerIndex,
	})
	if err != nil {
		return errors.Wrap(err, "vppapi WireguardPeersDump returned error")
	}
	defer func() { _ = dp.Close() }()

	details, err := dp.Recv()
	if err != nil {
		return errors.Wrapf(err, "error retrieving WireguardPeersDetails")
	}
	log.FromContext(ctx).
		WithField("peerIndex", peerIndex).
		WithField("details.Flags", details.Peer.Flags).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "WireguardPeersDump").Debug("completed")

	isEstablished := details.Peer.Flags & wireguard.WIREGUARD_PEER_ESTABLISHED
	if isEstablished != 0 {
		return nil
	}
	now = time.Now()
	for {
		select {
		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), "provided context is done")
		case rawMsg := <-notifCh:
			if msg, ok := rawMsg.(*wireguard.WireguardPeerEvent); ok &&
				msg.PeerIndex == peerIndex &&
				msg.Flags&wireguard.WIREGUARD_PEER_ESTABLISHED != 0 {
				log.FromContext(ctx).
					WithField("peerIndex", peerIndex).
					WithField("msg.Flags", msg.Flags).
					WithField("duration", time.Since(now)).
					WithField("vppapi", "WireguardPeerEvent").Debug("completed")
				return nil
			}
		}
	}
}

func getAPIChannel(ctx context.Context, vppConn Connection, peerIndex uint32) (api.Channel, error) {
	apiChannel, err := vppConn.NewAPIChannelBuffered(256, 256)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get new channel for communication with VPP via govpp core")
	}
	now := time.Now()
	if _, err = wireguard.NewServiceClient(vppConn).WantWireguardPeerEvents(ctx, &wireguard.WantWireguardPeerEvents{
		SwIfIndex:     interface_types.InterfaceIndex(^uint32(0)),
		PeerIndex:     peerIndex,
		EnableDisable: 1,
		PID:           uint32(os.Getpid()),
	}); err != nil {
		apiChannel.Close()
		return nil, errors.Wrap(err, "vppapi WantWireguardPeerEvents returned error")
	}
	log.FromContext(ctx).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "WantWireguardPeerEvents").Info("completed")

	go func() {
		<-ctx.Done()
		apiChannel.Close()
	}()
	return apiChannel, nil
}
