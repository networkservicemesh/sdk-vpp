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
	connectionID, networkService, nsc, nscInterface, nseInterface string, isClose bool,
) {
	if prometheus.IsEnabled() && isClose {
		if isClient {
			clientRxBytes.delete([]string{connectionID, networkService, nsc, nscInterface, nseInterface})
			clientTxBytes.delete([]string{connectionID, networkService, nsc, nscInterface, nseInterface})
			clientRxPackets.delete([]string{connectionID, networkService, nsc, nscInterface, nseInterface})
			clientTxPackets.delete([]string{connectionID, networkService, nsc, nscInterface, nseInterface})
			clientDrops.delete([]string{connectionID, networkService, nsc, nscInterface, nseInterface})
		} else {
			serverRxBytes.delete([]string{connectionID, networkService, nsc, nscInterface, nseInterface})
			serverTxBytes.delete([]string{connectionID, networkService, nsc, nscInterface, nseInterface})
			serverRxPackets.delete([]string{connectionID, networkService, nsc, nscInterface, nseInterface})
			serverTxPackets.delete([]string{connectionID, networkService, nsc, nscInterface, nseInterface})
			serverDrops.delete([]string{connectionID, networkService, nsc, nscInterface, nseInterface})
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
				serverRxBytes.update([]string{connectionID, networkService, nsc, nscInterface, nseInterface}, float64(iface.Rx.Bytes))
				serverTxBytes.update([]string{connectionID, networkService, nsc, nscInterface, nseInterface}, float64(iface.Tx.Bytes))
				serverRxPackets.update([]string{connectionID, networkService, nsc, nscInterface, nseInterface}, float64(iface.Rx.Packets))
				serverTxPackets.update([]string{connectionID, networkService, nsc, nscInterface, nseInterface}, float64(iface.Tx.Packets))
				serverDrops.update([]string{connectionID, networkService, nsc, nscInterface, nseInterface}, float64(iface.Drops))
			} else {
				clientRxBytes.update([]string{connectionID, networkService, nsc, nscInterface, nseInterface}, float64(iface.Rx.Bytes))
				clientTxBytes.update([]string{connectionID, networkService, nsc, nscInterface, nseInterface}, float64(iface.Tx.Bytes))
				clientRxPackets.update([]string{connectionID, networkService, nsc, nscInterface, nseInterface}, float64(iface.Rx.Packets))
				clientTxPackets.update([]string{connectionID, networkService, nsc, nscInterface, nseInterface}, float64(iface.Tx.Packets))
				clientDrops.update([]string{connectionID, networkService, nsc, nscInterface, nseInterface}, float64(iface.Drops))
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
