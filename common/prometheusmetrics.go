package common

import "github.com/prometheus/client_golang/prometheus"

// PrometheusMetrics is a list of collected Prometheus Counters and Guages that will be exported
type PrometheusMetrics struct {
	LatestBlockCounter prometheus.Counter
}

func GetPrometheusMetrics() *PrometheusMetrics {
	m := &PrometheusMetrics{}
	m.LatestBlockCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "lightwalletd_get_latest_block",
		Help: "Number of times GetLatestBlock was called",
	})

	return m
}
