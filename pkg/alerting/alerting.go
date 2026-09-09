// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package alerting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Manager handles alerting
type Manager struct {
	mu          sync.RWMutex
	rules       map[string]*AlertRule
	webhooks    map[string]*WebhookConfig
	alertBuffer chan *Alert
	webhookSem  chan struct{}  // bounds concurrent webhook deliveries
	wg          sync.WaitGroup // tracks in-flight webhook goroutines
	sendMu      sync.Mutex     // guards alertBuffer sends vs Close
	closed      bool
	stopOnce    sync.Once
}

// AlertRule represents an alert rule
type AlertRule struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	Enabled       bool              `json:"enabled"`
	RuleType      string            `json:"rule_type"`
	Condition     string            `json:"condition"`
	Threshold     float64           `json:"threshold_value"`
	TimeWindow    int               `json:"time_window"`
	Severity      string            `json:"severity"`
	Actions       []string          `json:"actions"`
	Cooldown      int               `json:"cooldown_seconds"`
	LastTriggered time.Time         `json:"last_triggered"`
	CreatedBy     string            `json:"created_by"`
	Metadata      map[string]string `json:"metadata"`
}

// WebhookConfig represents a webhook configuration
type WebhookConfig struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	URL       string            `json:"url"`
	Secret    string            `json:"secret"`
	Events    []string          `json:"events"`
	Enabled   bool              `json:"enabled"`
	Headers   map[string]string `json:"headers"`
	CreatedAt time.Time         `json:"created_at"`
}

// Alert represents an alert
type Alert struct {
	RuleID    string                 `json:"rule_id"`
	Severity  string                 `json:"severity"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details"`
	Timestamp time.Time              `json:"timestamp"`
}

// NewManager creates a new alerting manager
func NewManager() *Manager {
	m := &Manager{
		rules:       make(map[string]*AlertRule),
		webhooks:    make(map[string]*WebhookConfig),
		alertBuffer: make(chan *Alert, 1000),
		webhookSem:  make(chan struct{}, 16), // cap concurrent webhook deliveries
	}
	m.wg.Add(1)
	go m.processAlerts()
	return m
}

// Close stops the background alert processor and waits for in-flight webhook
// deliveries to finish. It is safe to call multiple times.
func (m *Manager) Close() {
	m.stopOnce.Do(func() {
		m.sendMu.Lock()
		m.closed = true
		close(m.alertBuffer)
		m.sendMu.Unlock()
	})
	m.wg.Wait()
}

// AddRule adds an alert rule
func (m *Manager) AddRule(rule *AlertRule) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rules[rule.ID] = rule
}

// RemoveRule removes an alert rule
func (m *Manager) RemoveRule(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.rules, id)
}

// GetRules returns all alert rules
func (m *Manager) GetRules() []*AlertRule {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rules := make([]*AlertRule, 0, len(m.rules))
	for _, rule := range m.rules {
		rules = append(rules, rule)
	}
	return rules
}

// AddWebhook adds a webhook configuration
func (m *Manager) AddWebhook(webhook *WebhookConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.webhooks[webhook.ID] = webhook
}

// RemoveWebhook removes a webhook
func (m *Manager) RemoveWebhook(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.webhooks, id)
}

// GetWebhooks returns all webhooks
func (m *Manager) GetWebhooks() []*WebhookConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	webhooks := make([]*WebhookConfig, 0, len(m.webhooks))
	for _, webhook := range m.webhooks {
		webhooks = append(webhooks, webhook)
	}
	return webhooks
}

// TriggerAlert triggers an alert
func (m *Manager) TriggerAlert(alert *Alert) {
	m.sendMu.Lock()
	if m.closed {
		m.sendMu.Unlock()
		return
	}
	select {
	case m.alertBuffer <- alert:
		m.sendMu.Unlock()
	default:
		m.sendMu.Unlock()
		log.Printf("WARNING: Alert buffer full, dropping alert")
	}
}

// processAlerts processes buffered alerts
func (m *Manager) processAlerts() {
	defer m.wg.Done()

	for alert := range m.alertBuffer {
		m.sendAlert(alert)
	}
}

// sendAlert sends an alert to all configured destinations
func (m *Manager) sendAlert(alert *Alert) {
	// Snapshot the webhook configurations (by value) under the read lock so the
	// goroutines below never race with AddWebhook/RemoveWebhook mutating the
	// shared *WebhookConfig values.
	m.mu.RLock()
	webhooks := make([]WebhookConfig, 0, len(m.webhooks))
	for _, webhook := range m.webhooks {
		if webhook.Enabled {
			webhooks = append(webhooks, *webhook)
		}
	}
	m.mu.RUnlock()

	for i := range webhooks {
		wh := webhooks[i] // local copy per goroutine
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			// Bound concurrent deliveries so a flood of alerts can't spawn an
			// unbounded number of goroutines.
			m.webhookSem <- struct{}{}
			defer func() { <-m.webhookSem }()
			m.sendWebhook(&wh, alert)
		}()
	}
}

// sendWebhook sends an alert to a webhook
func (m *Manager) sendWebhook(webhook *WebhookConfig, alert *Alert) {
	payload := map[string]interface{}{
		"timestamp": alert.Timestamp,
		"severity":  alert.Severity,
		"message":   alert.Message,
		"details":   alert.Details,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("ERROR: Failed to marshal webhook payload: %v", err)
		return
	}

	if err := validateWebhookURL(webhook.URL); err != nil {
		log.Printf("ERROR: Refusing to send webhook: %v", err)
		return
	}

	req, err := http.NewRequest("POST", webhook.URL, bytes.NewBuffer(data))
	if err != nil {
		log.Printf("ERROR: Failed to create webhook request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range webhook.Headers {
		req.Header.Set(k, v)
	}

	if webhook.Secret != "" {
		req.Header.Set("X-Webhook-Secret", webhook.Secret)
	}

	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: webhookTransport(),
		// Refuse redirects: a redirect could send the payload to an internal
		// address that validateWebhookURL already rejected.
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return fmt.Errorf("webhook redirects are not followed")
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("ERROR: Failed to send webhook: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("WARNING: Webhook returned non-success status: %d", resp.StatusCode)
	}
}

// validateWebhookURL rejects webhook targets that are not plain HTTP(S) or that
// resolve to an address inside the host's own network, which would turn the
// alerting subsystem into an SSRF proxy.
func validateWebhookURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid webhook URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported webhook scheme %q", u.Scheme)
	}

	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("webhook URL has no host")
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("cannot resolve webhook host %q: %w", host, err)
	}
	for _, ip := range ips {
		if isBlockedWebhookIP(ip) {
			return fmt.Errorf("webhook host %q resolves to blocked address %s", host, ip)
		}
	}
	return nil
}

// isBlockedWebhookIP reports whether ip is loopback, private, link-local (which
// covers the cloud metadata endpoint), multicast, unspecified, or carrier-grade
// NAT space.
func isBlockedWebhookIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() || ip.IsMulticast() {
		return true
	}
	if v4 := ip.To4(); v4 != nil && v4[0] == 100 && v4[1]&0xc0 == 64 {
		return true // 100.64.0.0/10
	}
	return false
}

// EvaluateMetric evaluates a metric against alert rules
func (m *Manager) EvaluateMetric(metricName string, value float64, labels map[string]string) {
	// Use the write lock: triggering a rule mutates rule.LastTriggered, so an
	// RLock here would race with concurrent evaluators.
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, rule := range m.rules {
		if !rule.Enabled {
			continue
		}

		// Check cooldown
		if time.Since(rule.LastTriggered) < time.Duration(rule.Cooldown)*time.Second {
			continue
		}

		// Simple threshold evaluation
		if rule.RuleType == "threshold" {
			shouldTrigger := false
			if rule.Condition == "greater_than" && value > rule.Threshold {
				shouldTrigger = true
			} else if rule.Condition == "less_than" && value < rule.Threshold {
				shouldTrigger = true
			} else if rule.Condition == "equals" && value == rule.Threshold {
				shouldTrigger = true
			}

			if shouldTrigger {
				rule.LastTriggered = time.Now()
				alert := &Alert{
					RuleID:   rule.ID,
					Severity: rule.Severity,
					Message:  rule.Name,
					Details: map[string]interface{}{
						"metric":    metricName,
						"value":     value,
						"threshold": rule.Threshold,
						"condition": rule.Condition,
						"labels":    labels,
					},
					Timestamp: time.Now(),
				}
				m.TriggerAlert(alert)
			}
		}
	}
}

// webhookTransport builds a transport that re-checks the address the
// connection actually goes to. validateWebhookURL resolves the host up front,
// but the name can resolve differently by the time the dial happens (DNS
// rebinding), so the decisive check belongs here, after the address is known
// and before the socket is used.
func webhookTransport() *http.Transport {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}

	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			if ip := net.ParseIP(host); ip != nil && isBlockedWebhookIP(ip) {
				return nil, fmt.Errorf("webhook target resolves to blocked address %s", ip)
			}
			return dialer.DialContext(ctx, network, addr)
		},
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 20 * time.Second,
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
	}
}
