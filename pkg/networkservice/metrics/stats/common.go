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

var (
	lastClientRxBytes   uint64
	lastClientTxBytes   uint64
	lastClientRxPackets uint64
	lastClientTxPackets uint64
	lastClientDrops     uint64
	lastServerRxBytes   uint64
	lastServerTxBytes   uint64
	lastServerRxPackets uint64
	lastServerTxPackets uint64
	lastServerDrops     uint64
)

func updateClientRxBytes(current uint64) {
	if current < lastClientRxBytes {
		lastClientRxBytes = 0
	}
	delta := current - lastClientRxBytes
	ClientRxBytes.Add(float64(delta))
	lastClientRxBytes = current
}

func updateClientTxBytes(current uint64) {
	if current < lastClientTxBytes {
		lastClientTxBytes = 0
	}
	delta := current - lastClientTxBytes
	ClientTxBytes.Add(float64(delta))
	lastClientTxBytes = current
}

func updateClientRxPackets(current uint64) {
	if current < lastClientRxPackets {
		lastClientRxPackets = 0
	}
	delta := current - lastClientRxPackets
	ClientRxPackets.Add(float64(delta))
	lastClientRxPackets = current
}

func updateClientTxPackets(current uint64) {
	if current < lastClientTxPackets {
		lastClientTxPackets = 0
	}
	delta := current - lastClientTxPackets
	ClientTxPackets.Add(float64(delta))
	lastClientTxPackets = current
}

func updateClientDrops(current uint64) {
	if current < lastClientDrops {
		lastClientDrops = 0
	}
	delta := current - lastClientDrops
	ClientDrops.Add(float64(delta))
	lastClientDrops = current
}

func updateServerRxBytes(current uint64) {
	if current < lastServerRxBytes {
		lastServerRxBytes = 0
	}
	delta := current - lastServerRxBytes
	ServerRxBytes.Add(float64(delta))
	lastServerRxBytes = current
}

func updateServerTxBytes(current uint64) {
	if current < lastServerTxBytes {
		lastServerTxBytes = 0
	}
	delta := current - lastServerTxBytes
	ServerTxBytes.Add(float64(delta))
	lastServerTxBytes = current
}

func updateServerRxPackets(current uint64) {
	if current < lastServerRxPackets {
		lastServerRxPackets = 0
	}
	delta := current - lastServerRxPackets
	ServerRxPackets.Add(float64(delta))
	lastServerRxPackets = current
}

func updateServerTxPackets(current uint64) {
	if current < lastServerTxPackets {
		lastServerTxPackets = 0
	}
	delta := current - lastServerTxPackets
	ServerTxPackets.Add(float64(delta))
	lastServerTxPackets = current
}

func updateServerDrops(current uint64) {
	if current < lastServerDrops {
		lastServerDrops = 0
	}
	delta := current - lastServerDrops
	ServerDrops.Add(float64(delta))
	lastServerDrops = current
}

// Save retrieved vpp interface metrics in pathSegment
func retrieveMetrics(ctx context.Context, statsConn *core.StatsConnection, segment *networkservice.PathSegment, isClient bool) {
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

		if prometheus.IsEnabled() {
			if addName == serverPref {
				updateServerRxBytes(iface.Rx.Bytes)
				updateServerTxBytes(iface.Tx.Bytes)
				updateServerRxPackets(iface.Rx.Packets)
				updateServerTxPackets(iface.Tx.Packets)
				updateServerDrops(iface.Drops)
			} else {
				updateClientRxBytes(iface.Rx.Bytes)
				updateClientTxBytes(iface.Tx.Bytes)
				updateClientRxPackets(iface.Rx.Packets)
				updateClientTxPackets(iface.Tx.Packets)
				updateClientDrops(iface.Drops)
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
