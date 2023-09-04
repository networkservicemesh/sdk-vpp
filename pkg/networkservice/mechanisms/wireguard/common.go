// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
//
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

package wireguard

import (
	"context"
	"time"

	"github.com/networkservicemesh/govpp/binapi/wireguard"
	"github.com/pkg/errors"
	"go.fd.io/govpp/api"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	wireguardMech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/wireguard"
	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/types"
)

// createInterface - returns public key of wireguard interface
func createInterface(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection, privateKey wgtypes.Key, isClient bool) (string, error) {
	if mechanism := wireguardMech.ToMechanism(conn.GetMechanism()); mechanism != nil {
		if pubKeyStr, ok := load(ctx, isClient); ok {
			return pubKeyStr, nil
		}

		now := time.Now()
		wgIfCreate := &wireguard.WireguardInterfaceCreate{
			Interface: wireguard.WireguardInterface{
				UserInstance: ^uint32(0),
				PrivateKey:   privateKey[:],
				Port:         mechanism.SrcPort(),
				SrcIP:        types.ToVppAddress(mechanism.SrcIP()),
			},
			GenerateKey: false,
		}
		if !isClient {
			wgIfCreate.Interface.Port = mechanism.DstPort()
			wgIfCreate.Interface.SrcIP = types.ToVppAddress(mechanism.DstIP())
		}

		rspIf, err := wireguard.NewServiceClient(vppConn).WireguardInterfaceCreate(ctx, wgIfCreate)
		if err != nil {
			return "", errors.Wrap(err, "vppapi WireguardInterfaceCreate returned error")
		}
		log.FromContext(ctx).
			WithField("swIfIndex", rspIf.SwIfIndex).
			WithField("SrcAddress", wgIfCreate.Interface.SrcIP).
			WithField("Port", wgIfCreate.Interface.Port).
			WithField("duration", time.Since(now)).
			WithField("vppapi", "WireguardInterfaceCreate").Debug("completed")
		ifindex.Store(ctx, isClient, rspIf.SwIfIndex)

		newPublicKey := privateKey.PublicKey().String()
		store(ctx, newPublicKey, isClient)
		return newPublicKey, nil
	}
	return "", nil
}

func delInterface(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection, isClient bool) error {
	if mechanism := wireguardMech.ToMechanism(conn.GetMechanism()); mechanism != nil {
		if _, ok := loadAndDelete(ctx, isClient); !ok {
			return nil
		}

		swIfIndex, ok := ifindex.LoadAndDelete(ctx, isClient)
		if !ok {
			return nil
		}
		now := time.Now()
		wgIfDel := &wireguard.WireguardInterfaceDelete{
			SwIfIndex: swIfIndex,
		}

		_, err := wireguard.NewServiceClient(vppConn).WireguardInterfaceDelete(ctx, wgIfDel)
		if err != nil {
			return errors.Wrap(err, "vppapi WireguardInterfaceDelete returned error")
		}
		log.FromContext(ctx).
			WithField("swIfIndex", wgIfDel.SwIfIndex).
			WithField("duration", time.Since(now)).
			WithField("vppapi", "WireguardInterfaceDelete").Debug("completed")
	}
	return nil
}
