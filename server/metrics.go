package server

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// ServerUptime tracks how long the server has been running
	// Includes start_time parameter to track different server runs
	ServerUptime = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "server_uptime",
		Help: "Current uptime of the server in seconds",
	}, []string{"start_time"})

	// RequestCount tracks the number of requests by method and status
	RequestCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "request_count",
		Help: "Total number of requests",
	}, []string{"method", "status", "retry"})

	// RequestDuration tracks how long requests take
	RequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "request_duration",
		Help: "Duration of requests in seconds",
	}, []string{"method", "status", "retry"})

	// RetryCount tracks the number of retry attempts
	RetryCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "retry_count",
		Help: "Total number of retry attempts",
	}, []string{"method", "status"})
)

// MetricsUpdater handles updating metrics over time
type MetricsUpdater struct {
	startTime time.Time
}

// NewMetricsUpdater creates a new metrics updater
func NewMetricsUpdater() *MetricsUpdater {
	startTime := time.Now()

	// Set initial uptime with start time parameter
	ServerUptime.WithLabelValues(startTime.Format(time.RFC3339)).Set(0)

	return &MetricsUpdater{
		startTime: startTime,
	}
}

// UpdateUptime updates the uptime metric
func (m *MetricsUpdater) UpdateUptime() {
	if m == nil {
		return
	}

	uptime := time.Since(m.startTime).Seconds()
	ServerUptime.WithLabelValues(m.startTime.Format(time.RFC3339)).Set(uptime)
}

// GetStartTime returns the server start time
func (m *MetricsUpdater) GetStartTime() time.Time {
	if m == nil {
		return time.Time{}
	}
	return m.startTime
}
