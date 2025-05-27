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

package stats

import (
	"os"
	"strings"
	"sync"

	prom "github.com/networkservicemesh/sdk/pkg/tools/prometheus"
	"github.com/prometheus/client_golang/prometheus"
)

type labeledCounter struct {
	metric     *prometheus.CounterVec
	lastValues map[string]float64
	mu         sync.Mutex
}

func newLabeledCounter(opts prometheus.CounterOpts, labelNames []string) *labeledCounter {
	vec := prometheus.NewCounterVec(opts, labelNames)
	prometheus.MustRegister(vec)

	return &labeledCounter{
		metric:     vec,
		lastValues: make(map[string]float64),
	}
}

func keyFromLabels(labelValues []string) string {
	return strings.Join(labelValues, "|")
}

func (lc *labeledCounter) update(labelValues []string, newValue float64) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	key := keyFromLabels(labelValues)
	delta := newValue - lc.lastValues[key]

	if delta > 0 {
		lc.metric.WithLabelValues(labelValues...).Add(delta)
		lc.lastValues[key] = newValue
	}
}

func (lc *labeledCounter) delete(labelValues []string) bool {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	delete(lc.lastValues, keyFromLabels(labelValues))
	return lc.metric.DeleteLabelValues(labelValues...)
}

var (
	prometheusInitOnce sync.Once

	clientRxBytes   *labeledCounter
	clientTxBytes   *labeledCounter
	clientRxPackets *labeledCounter
	clientTxPackets *labeledCounter
	clientDrops     *labeledCounter
	serverRxBytes   *labeledCounter
	serverTxBytes   *labeledCounter
	serverRxPackets *labeledCounter
	serverTxPackets *labeledCounter
	serverDrops     *labeledCounter
)

func registerMetrics() {
	if prom.IsEnabled() {
		forwarderName := os.Getenv("NSM_NAME")
		prefix := os.Getenv("PROMETHEUS_METRICS_PREFIX")
		if prefix != "" {
			prefix += "_"
		}
		clientRxBytes = newLabeledCounter(prometheus.CounterOpts{
			Name: prefix + forwarderName + "_client_rx_bytes_total",
			Help: "Total number of received bytes by the NetworkServiceClient vpp interface.",
		},
			[]string{"connection_id", "network_service", "nsc", "nsc_interface", "nse_interface"},
		)
		clientTxBytes = newLabeledCounter(prometheus.CounterOpts{
			Name: prefix + forwarderName + "_client_tx_bytes_total",
			Help: "Total number of transmitted bytes by the NetworkServiceClient vpp interface.",
		},
			[]string{"connection_id", "network_service", "nsc", "nsc_interface", "nse_interface"},
		)
		clientRxPackets = newLabeledCounter(prometheus.CounterOpts{
			Name: prefix + forwarderName + "_client_rx_packets_total",
			Help: "Total number of received packets by the NetworkServiceClient vpp interface.",
		},
			[]string{"connection_id", "network_service", "nsc", "nsc_interface", "nse_interface"},
		)
		clientTxPackets = newLabeledCounter(prometheus.CounterOpts{
			Name: prefix + forwarderName + "_client_tx_packets_total",
			Help: "Total number of transmitted packets by the NetworkServiceClient vpp interface.",
		},
			[]string{"connection_id", "network_service", "nsc", "nsc_interface", "nse_interface"},
		)
		clientDrops = newLabeledCounter(prometheus.CounterOpts{
			Name: prefix + forwarderName + "_client_drops_total",
			Help: "Total number of dropped packets by the NetworkServiceClient vpp interface.",
		},
			[]string{"connection_id", "network_service", "nsc", "nsc_interface", "nse_interface"},
		)
		serverRxBytes = newLabeledCounter(prometheus.CounterOpts{
			Name: prefix + forwarderName + "_server_rx_bytes_total",
			Help: "Total number of received bytes by the NetworkServiceServer vpp interface.",
		},
			[]string{"connection_id", "network_service", "nsc", "nsc_interface", "nse_interface"},
		)
		serverTxBytes = newLabeledCounter(prometheus.CounterOpts{
			Name: prefix + forwarderName + "_server_tx_bytes_total",
			Help: "Total number of transmitted bytes by the NetworkServiceServer vpp interface.",
		},
			[]string{"connection_id", "network_service", "nsc", "nsc_interface", "nse_interface"},
		)
		serverRxPackets = newLabeledCounter(prometheus.CounterOpts{
			Name: prefix + forwarderName + "_server_rx_packets_total",
			Help: "Total number of received packets by the NetworkServiceServer vpp interface.",
		},
			[]string{"connection_id", "network_service", "nsc", "nsc_interface", "nse_interface"},
		)
		serverTxPackets = newLabeledCounter(prometheus.CounterOpts{
			Name: prefix + forwarderName + "_server_tx_packets_total",
			Help: "Total number of transmitted packets by the NetworkServiceServer vpp interface.",
		},
			[]string{"connection_id", "network_service", "nsc", "nsc_interface", "nse_interface"},
		)
		serverDrops = newLabeledCounter(prometheus.CounterOpts{
			Name: prefix + forwarderName + "_server_drops_total",
			Help: "Total number of dropped packets by the NetworkServiceServer vpp interface.",
		},
			[]string{"connection_id", "network_service", "nsc", "nsc_interface", "nse_interface"},
		)
	}
}
