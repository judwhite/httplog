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
	httpRequestSizeBytes = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_size_bytes",
			Help: "The HTTP request sizes in bytes.",
		},
		[]string{"code", "handler", "method"},
	)
	httpResponseSizeBytes = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_response_size_bytes",
			Help: "The HTTP response sizes in bytes.",
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
	// # HELP prometheus_build_info A metric with a constant '1' value labeled by version, revision, branch, and goversion from which prometheus was built.
	// # TYPE prometheus_build_info gauge
	// prometheus_build_info{branch="master",goversion="go1.7.5",revision="bd1182d29f462c39544f94cc822830e1c64cf55b",version="1.5.2"} 1

	// # HELP prometheus_config_last_reload_success_timestamp_seconds Timestamp of the last successful configuration reload.
	// # TYPE prometheus_config_last_reload_success_timestamp_seconds gauge
	// prometheus_config_last_reload_success_timestamp_seconds 1.487454336e+09

	// # HELP prometheus_config_last_reload_successful Whether the last configuration reload attempt was successful.
	// # TYPE prometheus_config_last_reload_successful gauge
	// prometheus_config_last_reload_successful 1

	// # HELP prometheus_engine_queries The current number of queries being executed or waiting.
	// # TYPE prometheus_engine_queries gauge
	// prometheus_engine_queries 0

	// # HELP prometheus_local_storage_checkpoint_last_duration_seconds The duration in seconds it took to last checkpoint open chunks and chunks yet to be persisted.
	// # TYPE prometheus_local_storage_checkpoint_last_duration_seconds gauge
	// prometheus_local_storage_checkpoint_last_duration_seconds 0.07150000000000001

	prometheus.MustRegister(httpRequestDurationCounter)
	prometheus.MustRegister(httpRequestSizeBytes)
	prometheus.MustRegister(httpResponseSizeBytes)
	prometheus.MustRegister(httpRequestsTotal)
}
