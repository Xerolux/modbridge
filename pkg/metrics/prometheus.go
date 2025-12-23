package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// PrometheusMetrics provides Prometheus-compatible metrics.
type PrometheusMetrics struct {
	// Request metrics
	requestsTotal *prometheus.CounterVec
	errorsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec

	// Connection metrics
	activeConnections *prometheus.GaugeVec
	connectionsTotal  prometheus.Counter

	// Throughput metrics
	bytesReceived *prometheus.CounterVec
	bytesSent     *prometheus.CounterVec

	// Pool metrics
	poolSize        *prometheus.GaugeVec
	poolIdleConns   *prometheus.GaugeVec
	poolActiveConns *prometheus.GaugeVec

	// Device metrics
	devicesTotal    prometheus.Gauge
	deviceRequests  *prometheus.CounterVec

	// System metrics
	uptimeSeconds prometheus.Gauge
}

// NewPrometheusMetrics creates Prometheus metrics collectors.
func NewPrometheusMetrics(namespace string) *PrometheusMetrics {
	if namespace == "" {
		namespace = "modbus_proxy"
	}

	return &PrometheusMetrics{
		requestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "requests_total",
				Help:      "Total number of Modbus requests processed",
			},
			[]string{"proxy_id", "proxy_name"},
		),

		errorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "errors_total",
				Help:      "Total number of errors",
			},
			[]string{"proxy_id", "proxy_name", "type"},
		),

		requestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "request_duration_seconds",
				Help:      "Request duration in seconds",
				Buckets:   []float64{.001, .002, .005, .01, .025, .05, .1, .25, .5, 1},
			},
			[]string{"proxy_id", "proxy_name"},
		),

		activeConnections: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "active_connections",
				Help:      "Number of active client connections",
			},
			[]string{"proxy_id", "proxy_name"},
		),

		connectionsTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "connections_total",
				Help:      "Total number of connections accepted",
			},
		),

		bytesReceived: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "bytes_received_total",
				Help:      "Total bytes received from clients",
			},
			[]string{"proxy_id", "proxy_name"},
		),

		bytesSent: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "bytes_sent_total",
				Help:      "Total bytes sent to clients",
			},
			[]string{"proxy_id", "proxy_name"},
		),

		poolSize: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "pool_size",
				Help:      "Total connections in pool",
			},
			[]string{"proxy_id", "proxy_name"},
		),

		poolIdleConns: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "pool_idle_connections",
				Help:      "Idle connections in pool",
			},
			[]string{"proxy_id", "proxy_name"},
		),

		poolActiveConns: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "pool_active_connections",
				Help:      "Active connections in pool",
			},
			[]string{"proxy_id", "proxy_name"},
		),

		devicesTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "devices_total",
				Help:      "Total number of tracked devices",
			},
		),

		deviceRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "device_requests_total",
				Help:      "Total requests per device",
			},
			[]string{"device_ip", "device_name"},
		),

		uptimeSeconds: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "uptime_seconds",
				Help:      "Application uptime in seconds",
			},
		),
	}
}

// RecordRequest records a request in Prometheus format.
func (pm *PrometheusMetrics) RecordRequest(proxyID, proxyName string, durationSec float64, bytesIn, bytesOut int64) {
	pm.requestsTotal.WithLabelValues(proxyID, proxyName).Inc()
	pm.requestDuration.WithLabelValues(proxyID, proxyName).Observe(durationSec)
	pm.bytesReceived.WithLabelValues(proxyID, proxyName).Add(float64(bytesIn))
	pm.bytesSent.WithLabelValues(proxyID, proxyName).Add(float64(bytesOut))
}

// RecordError records an error in Prometheus format.
func (pm *PrometheusMetrics) RecordError(proxyID, proxyName, errorType string) {
	pm.errorsTotal.WithLabelValues(proxyID, proxyName, errorType).Inc()
}

// SetActiveConnections sets the number of active connections.
func (pm *PrometheusMetrics) SetActiveConnections(proxyID, proxyName string, count int) {
	pm.activeConnections.WithLabelValues(proxyID, proxyName).Set(float64(count))
}

// RecordConnection records a new connection.
func (pm *PrometheusMetrics) RecordConnection() {
	pm.connectionsTotal.Inc()
}

// SetPoolStats sets connection pool statistics.
func (pm *PrometheusMetrics) SetPoolStats(proxyID, proxyName string, total, idle, active int) {
	pm.poolSize.WithLabelValues(proxyID, proxyName).Set(float64(total))
	pm.poolIdleConns.WithLabelValues(proxyID, proxyName).Set(float64(idle))
	pm.poolActiveConns.WithLabelValues(proxyID, proxyName).Set(float64(active))
}

// SetDeviceCount sets the total number of tracked devices.
func (pm *PrometheusMetrics) SetDeviceCount(count int) {
	pm.devicesTotal.Set(float64(count))
}

// RecordDeviceRequest records a request from a specific device.
func (pm *PrometheusMetrics) RecordDeviceRequest(deviceIP, deviceName string) {
	pm.deviceRequests.WithLabelValues(deviceIP, deviceName).Inc()
}

// SetUptime sets the application uptime.
func (pm *PrometheusMetrics) SetUptime(seconds float64) {
	pm.uptimeSeconds.Set(seconds)
}
