package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP метрики
	HttpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "calculator_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	HttpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "calculator_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	// Метрики калькулятора
	CalculatorOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "calculator_operations_total",
			Help: "Total number of calculator operations",
		},
		[]string{"type"}, // success, error
	)

	CalculatorVariablesCount = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "calculator_variables_count",
			Help: "Current number of stored variables",
		},
	)

	CalculatorHistorySize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "calculator_history_size",
			Help: "Current size of command history",
		},
	)

	// WebRTC метрики (если будет интеграция)
	ActiveWebRTCConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "webrtc_active_connections",
			Help: "Number of active WebRTC connections",
		},
	)
)

// UpdateMetrics - обновление метрик калькулятора
func UpdateCalculatorMetrics(varsCount, historySize int) {
	CalculatorVariablesCount.Set(float64(varsCount))
	CalculatorHistorySize.Set(float64(historySize))
}
