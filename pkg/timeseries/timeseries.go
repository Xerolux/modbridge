package timeseries

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"sync"
	"time"
)

// DataPoint represents a time-series data point
type DataPoint struct {
	ProxyID         string
	RegisterAddress int
	Value           float64
	RawValue        uint16
	Unit            string
	Quality         string
	Timestamp       time.Time
}

// Manager manages time-series data
type Manager struct {
	mu            sync.RWMutex
	data          map[string][]*DataPoint // key: proxyID:registerAddress
	maxPoints     int
	retention     time.Duration
	filePath      string
	autoFlush     bool
	flushInterval time.Duration
}

// NewManager creates a new time-series manager
func NewManager(maxPoints int, retention time.Duration, filePath string) *Manager {
	m := &Manager{
		data:          make(map[string][]*DataPoint),
		maxPoints:     maxPoints,
		retention:     retention,
		filePath:      filePath,
		autoFlush:     true,
		flushInterval: 5 * time.Minute,
	}
	go m.autoFlushLoop()
	go m.cleanupOldData()
	return m
}

// Add adds a data point
func (m *Manager) Add(point *DataPoint) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%d", point.ProxyID, point.RegisterAddress)
	series := m.data[key]
	series = append(series, point)

	// Keep only maxPoints
	if len(series) > m.maxPoints {
		series = series[len(series)-m.maxPoints:]
	}

	m.data[key] = series
}

// Query retrieves data points for a specific register
func (m *Manager) Query(proxyID string, registerAddress int, start, end time.Time) []*DataPoint {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s:%d", proxyID, registerAddress)
	series := m.data[key]

	result := make([]*DataPoint, 0)
	for _, point := range series {
		if (start.IsZero() || point.Timestamp.After(start)) &&
			(end.IsZero() || point.Timestamp.Before(end)) {
			result = append(result, point)
		}
	}

	return result
}

// GetLatest returns the latest data point
func (m *Manager) GetLatest(proxyID string, registerAddress int) *DataPoint {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s:%d", proxyID, registerAddress)
	series := m.data[key]

	if len(series) == 0 {
		return nil
	}

	return series[len(series)-1]
}

// GetAggregatedData returns aggregated statistics
func (m *Manager) GetAggregatedData(proxyID string, registerAddress int, start, end time.Time) map[string]interface{} {
	points := m.Query(proxyID, registerAddress, start, end)

	if len(points) == 0 {
		return nil
	}

	sum := 0.0
	min := points[0].Value
	max := points[0].Value
	count := len(points)

	for _, point := range points {
		sum += point.Value
		if point.Value < min {
			min = point.Value
		}
		if point.Value > max {
			max = point.Value
		}
	}

	avg := sum / float64(count)

	// Calculate standard deviation
	variance := 0.0
	for _, point := range points {
		diff := point.Value - avg
		variance += diff * diff
	}
	variance /= float64(count)
	stdDev := math.Sqrt(variance)

	return map[string]interface{}{
		"count":  count,
		"min":    min,
		"max":    max,
		"avg":    avg,
		"stddev": stdDev,
		"sum":    sum,
		"start":  start,
		"end":    end,
	}
}

// ExportCSV exports data to CSV format
func (m *Manager) ExportCSV(proxyID string, registerAddress int, start, end time.Time) (string, error) {
	points := m.Query(proxyID, registerAddress, start, end)

	if m.filePath != "" {
		fileName := fmt.Sprintf("%s_%d_%s.csv", proxyID, registerAddress, time.Now().Format("20060102_150405"))
		if m.filePath != "" {
			fileName = fmt.Sprintf("%s/%s", m.filePath, fileName)
		}

		file, err := os.Create(fileName)
		if err != nil {
			return "", err
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		defer writer.Flush()

		writer.Write([]string{"Timestamp", "Value", "RawValue", "Unit", "Quality"})

		for _, point := range points {
			writer.Write([]string{
				point.Timestamp.Format(time.RFC3339),
				fmt.Sprintf("%.6f", point.Value),
				fmt.Sprintf("%d", point.RawValue),
				point.Unit,
				point.Quality,
			})
		}

		return fileName, nil
	}

	return "", fmt.Errorf("file path not configured")
}

// cleanupOldData removes data older than retention period
func (m *Manager) cleanupOldData() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		cutoff := time.Now().Add(-m.retention)
		for key, series := range m.data {
			newSeries := make([]*DataPoint, 0)
			for _, point := range series {
				if point.Timestamp.After(cutoff) {
					newSeries = append(newSeries, point)
				}
			}
			if len(newSeries) > 0 {
				m.data[key] = newSeries
			} else {
				delete(m.data, key)
			}
		}
		m.mu.Unlock()
	}
}

// autoFlushLoop periodically flushes data to disk
func (m *Manager) autoFlushLoop() {
	if !m.autoFlush {
		return
	}

	ticker := time.NewTicker(m.flushInterval)
	defer ticker.Stop()

	for range ticker.C {
		m.Flush()
	}
}

// Flush flushes all data to disk
func (m *Manager) Flush() error {
	// Implement file-based persistence if needed
	return nil
}

// GetStats returns time-series statistics
func (m *Manager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalPoints := 0
	for _, series := range m.data {
		totalPoints += len(series)
	}

	return map[string]interface{}{
		"total_series": len(m.data),
		"total_points": totalPoints,
		"max_points":   m.maxPoints,
		"retention":    m.retention.String(),
	}
}
