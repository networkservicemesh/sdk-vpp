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
	"os"
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
			Help: "Total number of received bytes by the local NetworkServiceClient vpp interface.",
		},
	)
	// ClientTxBytes - Total transmitted bytes by client
	ClientTxBytes = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "client_tx_bytes_total",
			Help: "Total number of transmitted bytes by the local NetworkServiceClient vpp interface.",
		},
	)
	// ClientRxPackets - Total received packets by client
	ClientRxPackets = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "client_rx_packets_total",
			Help: "Total number of received packets by the local NetworkServiceClient vpp interface.",
		},
	)
	// ClientTxPackets - Total transmitted packets by client
	ClientTxPackets = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "client_tx_packets_total",
			Help: "Total number of transmitted packets by the local NetworkServiceClient vpp interface.",
		},
	)
	// ClientDrops - Total drops by client
	ClientDrops = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "client_drops_total",
			Help: "Total number of dropped packets by the local NetworkServiceClient vpp interface.",
		},
	)
	// ServerRxBytes - Total received bytes by server
	ServerRxBytes = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "server_rx_bytes_total",
			Help: "Total number of received bytes by the local NetworkServiceServer vpp interface.",
		},
	)
	// ServerTxBytes - Total transmitted bytes by server
	ServerTxBytes = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "server_tx_bytes_total",
			Help: "Total number of transmitted bytes by the local NetworkServiceServer vpp interface.",
		},
	)
	// ServerRxPackets - Total received packets by server
	ServerRxPackets = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "server_rx_packets_total",
			Help: "Total number of received packets by the local NetworkServiceServer vpp interface.",
		},
	)
	// ServerTxPackets - Total transmitted packets by server
	ServerTxPackets = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "server_tx_packets_total",
			Help: "Total number of transmitted packets by the local NetworkServiceServer vpp interface.",
		},
	)
	// ServerDrops - Total drops by server
	ServerDrops = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "server_drops_total",
			Help: "Total number of dropped packets by the local NetworkServiceServer vpp interface.",
		},
	)
)

func registerMetrics() {
	if prom.IsEnabled() {
		prometheusMetricPrefixEnv := "PROMETHEUS_METRIC_PREFIX"
		prefix := os.Getenv(prometheusMetricPrefixEnv)
		if prefix != "" {
			ClientRxBytes = prometheus.NewGauge(
				prometheus.GaugeOpts{
					Name: prefix + "client_rx_bytes_total",
					Help: "Total number of received bytes by the local NetworkServiceClient vpp interface.",
				},
			)
			ClientTxBytes = prometheus.NewGauge(
				prometheus.GaugeOpts{
					Name: prefix + "client_tx_bytes_total",
					Help: "Total number of transmitted bytes by the local NetworkServiceClient vpp interface.",
				},
			)
			ClientRxPackets = prometheus.NewGauge(
				prometheus.GaugeOpts{
					Name: prefix + "client_rx_packets_total",
					Help: "Total number of received packets by the local NetworkServiceClient vpp interface.",
				},
			)
			ClientTxPackets = prometheus.NewGauge(
				prometheus.GaugeOpts{
					Name: prefix + "client_tx_packets_total",
					Help: "Total number of transmitted packets by the local NetworkServiceClient vpp interface.",
				},
			)
			ClientDrops = prometheus.NewGauge(
				prometheus.GaugeOpts{
					Name: prefix + "client_drops_total",
					Help: "Total number of dropped packets by the local NetworkServiceClient vpp interface.",
				},
			)
			ServerRxBytes = prometheus.NewGauge(
				prometheus.GaugeOpts{
					Name: prefix + "server_rx_bytes_total",
					Help: "Total number of received bytes by the local NetworkServiceServer vpp interface.",
				},
			)
			ServerTxBytes = prometheus.NewGauge(
				prometheus.GaugeOpts{
					Name: prefix + "server_tx_bytes_total",
					Help: "Total number of transmitted bytes by the local NetworkServiceServer vpp interface.",
				},
			)
			ServerRxPackets = prometheus.NewGauge(
				prometheus.GaugeOpts{
					Name: prefix + "server_rx_packets_total",
					Help: "Total number of received packets by the local NetworkServiceServer vpp interface.",
				},
			)
			ServerTxPackets = prometheus.NewGauge(
				prometheus.GaugeOpts{
					Name: prefix + "server_tx_packets_total",
					Help: "Total number of transmitted packets by the local NetworkServiceServer vpp interface.",
				},
			)
			ServerDrops = prometheus.NewGauge(
				prometheus.GaugeOpts{
					Name: prefix + "server_drops_total",
					Help: "Total number of dropped packets by the local NetworkServiceServer vpp interface.",
				},
			)
		}

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
