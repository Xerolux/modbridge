package metrics

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// PrometheusExporter exports metrics in Prometheus format
type PrometheusExporter struct {
	mu       sync.RWMutex
	metrics  map[string]*Metric
	labels   map[string]string
	registry *MetricRegistry
}

// Metric represents a single metric
type Metric struct {
	Name      string
	Type      MetricType
	Value     float64
	Labels    map[string]string
	Help      string
	Timestamp time.Time
}

// MetricType defines the type of metric
type MetricType string

const (
	Counter   MetricType = "counter"
	Gauge     MetricType = "gauge"
	Histogram MetricType = "histogram"
	Summary   MetricType = "summary"
)

// NewPrometheusExporter creates a new Prometheus exporter
func NewPrometheusExporter(registry *MetricRegistry) *PrometheusExporter {
	return &PrometheusExporter{
		metrics:  make(map[string]*Metric),
		labels:   make(map[string]string),
		registry: registry,
	}
}

// SetLabel sets a global label
func (pe *PrometheusExporter) SetLabel(key, value string) {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	pe.labels[key] = value
}

// RecordMetric records a metric
func (pe *PrometheusExporter) RecordMetric(name string, metricType MetricType, value float64, labels map[string]string, help string) {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	// Merge global labels
	finalLabels := make(map[string]string)
	for k, v := range pe.labels {
		finalLabels[k] = v
	}
	for k, v := range labels {
		finalLabels[k] = v
	}

	pe.metrics[name] = &Metric{
		Name:      name,
		Type:      metricType,
		Value:     value,
		Labels:    finalLabels,
		Help:      help,
		Timestamp: time.Now(),
	}
}

// Increment increments a counter metric
func (pe *PrometheusExporter) Increment(name string, value float64, labels map[string]string) {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	key := pe.metricKey(name, labels)

	if existing, exists := pe.metrics[key]; exists {
		existing.Value += value
		existing.Timestamp = time.Now()
	} else {
		pe.metrics[key] = &Metric{
			Name:      name,
			Type:      Counter,
			Value:     value,
			Labels:    labels,
			Timestamp: time.Now(),
		}
	}
}

// Set sets a gauge metric
func (pe *PrometheusExporter) Set(name string, value float64, labels map[string]string) {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	key := pe.metricKey(name, labels)

	pe.metrics[key] = &Metric{
		Name:      name,
		Type:      Gauge,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
	}
}

// metricKey generates a unique key for a metric with labels
func (pe *PrometheusExporter) metricKey(name string, labels map[string]string) string {
	if len(labels) == 0 {
		return name
	}

	// Sort labels for consistent key generation
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(labels)+1)
	parts = append(parts, name)
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, labels[k]))
	}

	return strings.Join(parts, "|")
}

// Export exports metrics in Prometheus format
func (pe *PrometheusExporter) Export() string {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	var builder strings.Builder

	// Group metrics by name
	grouped := make(map[string][]*Metric)
	for _, metric := range pe.metrics {
		grouped[metric.Name] = append(grouped[metric.Name], metric)
	}

	// Export each metric group
	for name, metrics := range grouped {
		if len(metrics) == 0 {
			continue
		}

		// Add HELP and TYPE
		if metrics[0].Help != "" {
			builder.WriteString(fmt.Sprintf("# HELP %s %s\n", name, metrics[0].Help))
		}
		builder.WriteString(fmt.Sprintf("# TYPE %s %s\n", name, metrics[0].Type))

		// Add metric lines
		for _, metric := range metrics {
			builder.WriteString(pe.formatMetric(metric))
		}

		builder.WriteString("\n")
	}

	return builder.String()
}

// formatMetric formats a single metric
func (pe *PrometheusExporter) formatMetric(metric *Metric) string {
	labelStr := ""
	if len(metric.Labels) > 0 {
		labels := make([]string, 0, len(metric.Labels))
		for k, v := range metric.Labels {
			labels = append(labels, fmt.Sprintf(`%s="%s"`, k, escapeLabelValue(v)))
		}
		labelStr = fmt.Sprintf("{%s}", strings.Join(labels, ","))
	}

	return fmt.Sprintf("%s%s %s\n", metric.Name, labelStr, formatFloat(metric.Value))
}

// escapeLabelValue escapes a label value
func escapeLabelValue(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	value = strings.ReplaceAll(value, "\n", "\\n")
	return value
}

// formatFloat formats a float value
func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// Handler returns an HTTP handler for the metrics endpoint
func (pe *PrometheusExporter) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(pe.Export()))
	}
}

// MetricRegistry stores all metrics
type MetricRegistry struct {
	mu      sync.RWMutex
	metrics map[string]*Metric
}

// NewMetricRegistry creates a new metric registry
func NewMetricRegistry() *MetricRegistry {
	return &MetricRegistry{
		metrics: make(map[string]*Metric),
	}
}

// Register registers a metric
func (mr *MetricRegistry) Register(name string, metricType MetricType, help string) {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	if _, exists := mr.metrics[name]; !exists {
		mr.metrics[name] = &Metric{
			Name: name,
			Type: metricType,
			Help: help,
		}
	}
}

// Get retrieves a metric
func (mr *MetricRegistry) Get(name string) (*Metric, bool) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	metric, exists := mr.metrics[name]
	return metric, exists
}

// GetAll returns all metrics
func (mr *MetricRegistry) GetAll() map[string]*Metric {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	all := make(map[string]*Metric)
	for k, v := range mr.metrics {
		all[k] = v
	}
	return all
}
