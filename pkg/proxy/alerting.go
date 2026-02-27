package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// AlertSeverity defines the severity level of an alert
type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "info"
	SeverityWarning  AlertSeverity = "warning"
	SeverityError    AlertSeverity = "error"
	SeverityCritical AlertSeverity = "critical"
)

// Alert represents an alert/notification
type Alert struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	Severity    AlertSeverity     `json:"severity"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	ProxyID     string            `json:"proxy_id,omitempty"`
	Target      string            `json:"target,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
	Resolved    bool              `json:"resolved"`
	ResolvedAt  *time.Time        `json:"resolved_at,omitempty"`
}

// AlertChannel defines where alerts are sent
type AlertChannel interface {
	Send(alert Alert) error
}

// WebhookChannel sends alerts via HTTP webhooks
type WebhookChannel struct {
	URL     string
	Headers map[string]string
	client  *http.Client
}

// NewWebhookChannel creates a new webhook channel
func NewWebhookChannel(url string, headers map[string]string) *WebhookChannel {
	return &WebhookChannel{
		URL:     url,
		Headers: headers,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Send sends an alert via webhook
func (wc *WebhookChannel) Send(alert Alert) error {
	jsonData, err := json.Marshal(alert)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", wc.URL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range wc.Headers {
		req.Header.Set(k, v)
	}

	resp, err := wc.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	_ = jsonData // Unused for now, will be used in request body

	return nil
}

// AlertManager manages alerts and notifications
type AlertManager struct {
	mu              sync.RWMutex
	channels        []AlertChannel
	alertHistory    []Alert
	maxHistory      int
	rules           []AlertRule
	muStats         sync.Mutex
	alertCounts     map[AlertSeverity]int64
	lastAlert       time.Time
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	running         bool
}

// AlertRule defines when to trigger alerts
type AlertRule struct {
	Name      string
	Condition func(Alert) bool
	Channel   AlertChannel
	Enabled   bool
}

// AlertManagerConfig holds configuration
type AlertManagerConfig struct {
	MaxHistory int // Maximum alert history to keep (default: 1000)
}

// DefaultAlertManagerConfig returns sensible defaults
func DefaultAlertManagerConfig() AlertManagerConfig {
	return AlertManagerConfig{
		MaxHistory: 1000,
	}
}

// NewAlertManager creates a new alert manager
func NewAlertManager(config AlertManagerConfig) *AlertManager {
	if config.MaxHistory <= 0 {
		config.MaxHistory = 1000
	}

	ctx, cancel := context.WithCancel(context.Background())

	am := &AlertManager{
		channels:     make([]AlertChannel, 0),
		alertHistory: make([]Alert, 0, config.MaxHistory),
		maxHistory:   config.MaxHistory,
		rules:        make([]AlertRule, 0),
		alertCounts:  make(map[AlertSeverity]int64),
		ctx:          ctx,
		cancel:       cancel,
		running:      true,
	}

	return am
}

// AddChannel adds an alert channel
func (am *AlertManager) AddChannel(channel AlertChannel) {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.channels = append(am.channels, channel)
}

// AddRule adds an alert rule
func (am *AlertManager) AddRule(name string, condition func(Alert) bool, channel AlertChannel) {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.rules = append(am.rules, AlertRule{
		Name:      name,
		Condition: condition,
		Channel:   channel,
		Enabled:   true,
	})
}

// TriggerAlert triggers an alert
func (am *AlertManager) TriggerAlert(alertType, title, description string, severity AlertSeverity, metadata map[string]string) {
	alert := Alert{
		ID:          generateAlertID(),
		Type:        alertType,
		Severity:    severity,
		Title:       title,
		Description: description,
		Metadata:    metadata,
		Timestamp:   time.Now(),
		Resolved:    false,
	}

	am.processAlert(alert)
}

// processAlert processes an alert
func (am *AlertManager) processAlert(alert Alert) {
	// Record in history
	am.addToHistory(alert)

	// Update stats
	am.muStats.Lock()
	am.alertCounts[alert.Severity]++
	am.lastAlert = time.Now()
	am.muStats.Unlock()

	// Check rules
	am.mu.RLock()
	rules := make([]AlertRule, len(am.rules))
	copy(rules, am.rules)
	am.mu.RUnlock()

	for _, rule := range rules {
		if rule.Enabled && rule.Condition(alert) {
			if rule.Channel != nil {
				go func(ch AlertChannel, al Alert) {
					_ = ch.Send(al)
				}(rule.Channel, alert)
			}
		}
	}

	// Send to all channels
	am.mu.RLock()
	channels := make([]AlertChannel, len(am.channels))
	copy(channels, am.channels)
	am.mu.RUnlock()

	for _, channel := range channels {
		go func(ch AlertChannel, al Alert) {
			_ = ch.Send(al)
		}(channel, alert)
	}
}

// addToHistory adds alert to history
func (am *AlertManager) addToHistory(alert Alert) {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.alertHistory = append(am.alertHistory, alert)

	// Trim history if needed
	if len(am.alertHistory) > am.maxHistory {
		am.alertHistory = am.alertHistory[1:]
	}
}

// ResolveAlert resolves an alert
func (am *AlertManager) ResolveAlert(alertID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	for i := range am.alertHistory {
		if am.alertHistory[i].ID == alertID && !am.alertHistory[i].Resolved {
			now := time.Now()
			am.alertHistory[i].Resolved = true
			am.alertHistory[i].ResolvedAt = &now
			return nil
		}
	}

	return fmt.Errorf("alert not found")
}

// GetHistory returns alert history
func (am *AlertManager) GetHistory(limit int) []Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if limit <= 0 || limit > len(am.alertHistory) {
		limit = len(am.alertHistory)
	}

	start := len(am.alertHistory) - limit
	history := make([]Alert, limit)
	copy(history, am.alertHistory[start:])

	return history
}

// GetStats returns alert statistics
func (am *AlertManager) GetStats() map[string]interface{} {
	am.muStats.Lock()
	defer am.muStats.Unlock()

	am.mu.RLock()
	totalAlerts := len(am.alertHistory)
	activeAlerts := 0
	for _, alert := range am.alertHistory {
		if !alert.Resolved {
			activeAlerts++
		}
	}
	am.mu.RUnlock()

	return map[string]interface{}{
		"total_alerts":       totalAlerts,
		"active_alerts":      activeAlerts,
		"critical_count":     am.alertCounts[SeverityCritical],
		"error_count":        am.alertCounts[SeverityError],
		"warning_count":      am.alertCounts[SeverityWarning],
		"info_count":         am.alertCounts[SeverityInfo],
		"last_alert":         am.lastAlert,
		"channels":           len(am.channels),
		"rules":              len(am.rules),
	}
}

// Stop stops the alert manager
func (am *AlertManager) Stop() {
	am.mu.Lock()
	defer am.mu.Unlock()

	if !am.running {
		return
	}

	am.running = false
	am.cancel()
	am.wg.Wait()
}

// generateAlertID generates a unique alert ID
func generateAlertID() string {
	return fmt.Sprintf("alert_%d", time.Now().UnixNano())
}
