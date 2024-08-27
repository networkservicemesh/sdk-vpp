// Copyright (c) 2024 Nordix Foundation.
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

package stats

import (
	"sync"

	prom "github.com/networkservicemesh/sdk/pkg/tools/prometheus"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	prometheusInitOnce sync.Once

	// ClientRxBytes - Total received bytes by client
	ClientRxBytes = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "client_rx_bytes_total",
			Help: "Total received bytes by client",
		},
	)
	// ClientTxBytes - Total transmitted bytes by client
	ClientTxBytes = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "client_tx_bytes_total",
			Help: "Total transmitted bytes by client",
		},
	)
	// ClientRxPackets - Total received packets by client
	ClientRxPackets = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "client_rx_packets_total",
			Help: "Total received packets by client",
		},
	)
	// ClientTxPackets - Total transmitted packets by client
	ClientTxPackets = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "client_tx_packets_total",
			Help: "Total transmitted packets by client",
		},
	)
	// ClientDrops - Total drops by client
	ClientDrops = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "client_drops_total",
			Help: "Total drops by client",
		},
	)
	// ServerRxBytes - Total received bytes by server
	ServerRxBytes = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "server_rx_bytes_total",
			Help: "Total received bytes by server",
		},
	)
	// ServerTxBytes - Total transmitted bytes by server
	ServerTxBytes = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "server_tx_bytes_total",
			Help: "Total transmitted bytes by server",
		},
	)
	// ServerRxPackets - Total received packets by server
	ServerRxPackets = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "server_rx_packets_total",
			Help: "Total received packets by server",
		},
	)
	// ServerTxPackets - Total transmitted packets by server
	ServerTxPackets = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "server_tx_packets_total",
			Help: "Total transmitted packets by server",
		},
	)
	// ServerDrops - Total drops by server
	ServerDrops = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "server_drops_total",
			Help: "Total drops by server",
		},
	)
)

func registerMetrics() {
	if prom.IsEnabled() {
		prometheus.MustRegister(ClientRxBytes)
		prometheus.MustRegister(ClientTxBytes)
		prometheus.MustRegister(ClientRxPackets)
		prometheus.MustRegister(ClientTxPackets)
		prometheus.MustRegister(ClientDrops)
		prometheus.MustRegister(ServerRxBytes)
		prometheus.MustRegister(ServerTxBytes)
		prometheus.MustRegister(ServerRxPackets)
		prometheus.MustRegister(ServerTxPackets)
		prometheus.MustRegister(ServerDrops)
	}
}
