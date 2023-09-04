// Copyright (c) 2022-2023 Cisco and/or its affiliates.
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

package memifrxmode

import (
	"context"
	"time"

	"go.fd.io/govpp/api"

	interfaces "github.com/networkservicemesh/govpp/binapi/interface"
	"github.com/networkservicemesh/govpp/binapi/interface_types"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

// Connection aggregates the api.Connection and api.ChannelProvider interfaces
type Connection interface {
	api.Connection
	api.ChannelProvider
}

func setRxMode(ctx context.Context, vppConn Connection, swIfIndex interface_types.InterfaceIndex) error {
	apiChannel, err := vppConn.NewAPIChannelBuffered(256, 256)
	if err != nil {
		return errors.Wrap(err, "failed to get new channel for communication with VPP via govpp core")
	}

	notifCh := make(chan api.Message, 256)
	subscription, err := apiChannel.SubscribeNotification(notifCh, &interfaces.SwInterfaceEvent{})
	if err != nil {
		return errors.Wrap(err, "failed to subscribe for receiving of the specified notification messages via provided Go channel")
	}

	go func() {
		defer apiChannel.Close()
		defer func() { _ = subscription.Unsubscribe() }()
		for {
			select {
			case <-ctx.Done():
				return
			case rawMsg := <-notifCh:
				if msg, ok := rawMsg.(*interfaces.SwInterfaceEvent); ok &&
					msg.SwIfIndex == swIfIndex &&
					msg.Flags&interface_types.IF_STATUS_API_FLAG_LINK_UP != 0 {
					now := time.Now()
					_, err = interfaces.NewServiceClient(vppConn).SwInterfaceSetRxMode(ctx, &interfaces.SwInterfaceSetRxMode{
						SwIfIndex: swIfIndex,
						Mode:      interface_types.RX_MODE_API_ADAPTIVE,
					})
					if err != nil {
						log.FromContext(ctx).
							WithField("swIfIndex", swIfIndex).
							WithField("mode", interface_types.RX_MODE_API_ADAPTIVE).
							WithField("duration", time.Since(now)).
							WithField("vppapi", "SwInterfaceSetRxMode").Debugf("error: %v", err.Error())
						return
					}
					log.FromContext(ctx).
						WithField("swIfIndex", swIfIndex).
						WithField("mode", interface_types.RX_MODE_API_ADAPTIVE).
						WithField("duration", time.Since(now)).
						WithField("vppapi", "SwInterfaceSetRxMode").Debug("completed")
					return
				}
			}
		}
	}()
	return nil
}
