// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package alerting

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

func TestValidateWebhookURLRejectsNonHTTPSchemes(t *testing.T) {
	for _, raw := range []string{
		"file:///etc/passwd",
		"gopher://example.com/",
		"ftp://example.com/hook",
		"://not-a-url",
	} {
		if err := validateWebhookURL(raw); err == nil {
			t.Errorf("validateWebhookURL(%q) = nil, want an error", raw)
		}
	}
}

func TestValidateWebhookURLRejectsInternalTargets(t *testing.T) {
	// The cloud metadata endpoint and the loopback interface are the targets
	// an SSRF is usually pointed at.
	for _, raw := range []string{
		"http://127.0.0.1:8080/hook",
		"http://localhost/hook",
		"http://169.254.169.254/latest/meta-data/",
		"http://10.0.0.5/hook",
		"http://192.168.1.1/hook",
		"http://172.16.0.1/hook",
		"http://[::1]/hook",
		"http://0.0.0.0/hook",
		"http://100.64.0.1/hook",
	} {
		err := validateWebhookURL(raw)
		if err == nil {
			t.Errorf("validateWebhookURL(%q) = nil, want it rejected", raw)
			continue
		}
		if !strings.Contains(err.Error(), "blocked") && !strings.Contains(err.Error(), "resolve") {
			t.Errorf("validateWebhookURL(%q) failed for the wrong reason: %v", raw, err)
		}
	}
}

func TestValidateWebhookURLRejectsMissingHost(t *testing.T) {
	if err := validateWebhookURL("http:///hook"); err == nil {
		t.Error("validateWebhookURL with no host = nil, want an error")
	}
}

func TestIsBlockedWebhookIP(t *testing.T) {
	blocked := []string{
		"127.0.0.1", "::1", "10.1.2.3", "172.20.0.1", "192.168.0.1",
		"169.254.169.254", "fe80::1", "224.0.0.1", "0.0.0.0",
		"100.64.0.1", "100.127.255.254",
	}
	for _, s := range blocked {
		if !isBlockedWebhookIP(net.ParseIP(s)) {
			t.Errorf("isBlockedWebhookIP(%s) = false, want true", s)
		}
	}

	allowed := []string{"93.184.216.34", "8.8.8.8", "2606:2800:220:1:248:1893:25c8:1946", "100.128.0.1", "99.255.255.255"}
	for _, s := range allowed {
		if isBlockedWebhookIP(net.ParseIP(s)) {
			t.Errorf("isBlockedWebhookIP(%s) = true, want false", s)
		}
	}
}

func TestWebhookTransportRejectsBlockedAddressAtDialTime(t *testing.T) {
	// The dial-time check is what closes the gap between resolving the name
	// and connecting to it.
	_, err := webhookTransport().DialContext(t.Context(), "tcp", "127.0.0.1:9")
	if err == nil {
		t.Fatal("dial to a loopback address succeeded, want it blocked")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Errorf("dial failed for the wrong reason: %v", err)
	}
}

func TestWebhookTransportAllowsPublicAddress(t *testing.T) {
	// The address is unroutable on purpose, so the dial cannot succeed. What
	// this asserts is that it fails on the network rather than on the guard,
	// which is why the context is cancelled quickly.
	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()

	_, err := webhookTransport().DialContext(ctx, "tcp", "203.0.113.1:9")
	if err != nil && strings.Contains(err.Error(), "blocked") {
		t.Errorf("public address was blocked: %v", err)
	}
}
