package httplog

import "github.com/prometheus/client_golang/prometheus"

var (
	httpRequestDurationCounter = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "The HTTP request latencies in seconds.",
		},
		[]string{"code", "handler", "method"},
	)
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests made.",
		},
		[]string{"code", "handler", "method"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestDurationCounter)
	prometheus.MustRegister(httpRequestsTotal)
}
