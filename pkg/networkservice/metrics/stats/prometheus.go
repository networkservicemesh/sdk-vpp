package stats

import (
	"github.com/networkservicemesh/sdk/pkg/tools/opentelemetry"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	ClientRxBytes = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "client_rx_bytes_total",
			Help: "Total received bytes by client",
		},
	)
	ClientTxBytes = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "client_tx_bytes_total",
			Help: "Total transmitted bytes by client",
		},
	)
	ClientRxPackets = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "client_rx_packets_total",
			Help: "Total received packets by client",
		},
	)
	ClientTxPackets = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "client_tx_packets_total",
			Help: "Total transmitted packets by client",
		},
	)
	ClientDrops = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "client_drops_total",
			Help: "Total drops by client",
		},
	)
	ServerRxBytes = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "server_rx_bytes_total",
			Help: "Total received bytes by server",
		},
	)
	ServerTxBytes = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "server_tx_bytes_total",
			Help: "Total transmitted bytes by server",
		},
	)

	ServerRxPackets = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "server_rx_packets_total",
			Help: "Total received packets by server",
		},
	)

	ServerTxPackets = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "server_tx_packets_total",
			Help: "Total transmitted packets by server",
		},
	)

	ServerDrops = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "server_drops_total",
			Help: "Total drops by server",
		},
	)
)

func init() {
	if opentelemetry.IsPrometheusEnabled() {
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
