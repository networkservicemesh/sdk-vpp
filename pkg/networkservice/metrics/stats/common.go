// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2022-2024 Cisco and/or its affiliates.
//
// Copyright (c) 2025 Nordix Foundation.
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

package stats

import (
	"context"
	"strconv"

	"go.fd.io/govpp/adapter"
	"go.fd.io/govpp/adapter/statsclient"
	"go.fd.io/govpp/api"
	"go.fd.io/govpp/core"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/prometheus"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
)

const serverPref string = "server_"

// Save retrieved vpp interface metrics in pathSegment
func retrieveMetrics(ctx context.Context, statsConn *core.StatsConnection, segment *networkservice.PathSegment, isClient bool,
	connectionId, networkService, nsc, nscInterface, nseInterface string, isClose bool,
) {
	if prometheus.IsEnabled() && isClose {
		if isClient {
			ClientRxBytes.Delete([]string{connectionId, networkService, nsc, nscInterface, nseInterface})
			ClientTxBytes.Delete([]string{connectionId, networkService, nsc, nscInterface, nseInterface})
			ClientRxPackets.Delete([]string{connectionId, networkService, nsc, nscInterface, nseInterface})
			ClientTxPackets.Delete([]string{connectionId, networkService, nsc, nscInterface, nseInterface})
			ClientDrops.Delete([]string{connectionId, networkService, nsc, nscInterface, nseInterface})
		} else {
			ServerRxBytes.Delete([]string{connectionId, networkService, nsc, nscInterface, nseInterface})
			ServerTxBytes.Delete([]string{connectionId, networkService, nsc, nscInterface, nseInterface})
			ServerRxPackets.Delete([]string{connectionId, networkService, nsc, nscInterface, nseInterface})
			ServerTxPackets.Delete([]string{connectionId, networkService, nsc, nscInterface, nseInterface})
			ServerDrops.Delete([]string{connectionId, networkService, nsc, nscInterface, nseInterface})
		}
	}

	swIfIndex, ok := ifindex.Load(ctx, isClient)
	if !ok {
		return
	}
	stats := new(api.InterfaceStats)
	if e := statsConn.GetInterfaceStats(stats); e != nil {
		log.FromContext(ctx).Errorf("getting interface stats failed:", e)
		return
	}

	addName := serverPref
	if isClient {
		addName = "client_"
	}
	for idx := range stats.Interfaces {
		iface := &stats.Interfaces[idx]
		if iface.InterfaceIndex != uint32(swIfIndex) {
			continue
		}

		if segment.Metrics == nil {
			segment.Metrics = make(map[string]string)
		}
		segment.Metrics[addName+"rx_bytes"] = strconv.FormatUint(iface.Rx.Bytes, 10)
		segment.Metrics[addName+"tx_bytes"] = strconv.FormatUint(iface.Tx.Bytes, 10)
		segment.Metrics[addName+"rx_packets"] = strconv.FormatUint(iface.Rx.Packets, 10)
		segment.Metrics[addName+"tx_packets"] = strconv.FormatUint(iface.Tx.Packets, 10)
		segment.Metrics[addName+"drops"] = strconv.FormatUint(iface.Drops, 10)

		if prometheus.IsEnabled() && !isClose {
			if addName == serverPref {
				ServerRxBytes.Update([]string{connectionId, networkService, nsc, nscInterface, nseInterface}, float64(iface.Rx.Bytes))
				ServerTxBytes.Update([]string{connectionId, networkService, nsc, nscInterface, nseInterface}, float64(iface.Tx.Bytes))
				ServerRxPackets.Update([]string{connectionId, networkService, nsc, nscInterface, nseInterface}, float64(iface.Rx.Packets))
				ServerTxPackets.Update([]string{connectionId, networkService, nsc, nscInterface, nseInterface}, float64(iface.Tx.Packets))
				ServerDrops.Update([]string{connectionId, networkService, nsc, nscInterface, nseInterface}, float64(iface.Drops))
			} else {
				ClientRxBytes.Update([]string{connectionId, networkService, nsc, nscInterface, nseInterface}, float64(iface.Rx.Bytes))
				ClientTxBytes.Update([]string{connectionId, networkService, nsc, nscInterface, nseInterface}, float64(iface.Tx.Bytes))
				ClientRxPackets.Update([]string{connectionId, networkService, nsc, nscInterface, nseInterface}, float64(iface.Rx.Packets))
				ClientTxPackets.Update([]string{connectionId, networkService, nsc, nscInterface, nseInterface}, float64(iface.Tx.Packets))
				ClientDrops.Update([]string{connectionId, networkService, nsc, nscInterface, nseInterface}, float64(iface.Drops))
			}
		}

		break
	}
}

func initFunc(chainCtx context.Context, statsSocket string) (*core.StatsConnection, error) {
	if statsSocket == "" {
		statsSocket = adapter.DefaultStatsSocket
	}
	statsConn, err := core.ConnectStats(statsclient.NewStatsClient(statsSocket))
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to Stats API")
	}
	go func() {
		<-chainCtx.Done()
		statsConn.Disconnect()
	}()
	return statsConn, nil
}
