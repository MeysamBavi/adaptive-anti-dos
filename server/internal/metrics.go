package internal

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

var (
	requestLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "A histogram of latencies for requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"ip"},
	)

	requestStatusCodes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_request_status_codes",
			Help: "Counter of status codes returned by HTTP server",
		},
		[]string{"code", "ip"},
	)
)

func init() {
	prometheus.MustRegister(requestLatency)
	prometheus.MustRegister(requestStatusCodes)
}

func ObserveRequestMetrics(start time.Time, statusCode int, ip string) {
	duration := time.Since(start)
	requestLatency.WithLabelValues(ip).Observe(duration.Seconds())
	requestStatusCodes.WithLabelValues(fmt.Sprint(statusCode), ip).Inc()
}
