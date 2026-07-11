// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

// Package updater provides GitHub-Release-based self-update functionality
// for the ModBridge binary, including version comparison, checksum
// verification, and atomic binary swapping.
package updater

import (
	"fmt"
	"strconv"
	"strings"
)

// CompareVersions reports whether latest is strictly newer than current.
// Both inputs may carry an optional leading "v" prefix. Versions are
// compared segment by segment as integers; missing trailing segments
// are treated as 0. The special values "" and "dev" for current always
// yield true (any real release is newer than a dev build).
func CompareVersions(current, latest string) (bool, error) {
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")

	// "dev" or empty current → any real release is newer
	if current == "" || current == "dev" {
		return true, nil
	}

	curParts, err := splitVersion(current)
	if err != nil {
		return false, fmt.Errorf("invalid current version %q: %w", current, err)
	}
	latParts, err := splitVersion(latest)
	if err != nil {
		return false, fmt.Errorf("invalid latest version %q: %w", latest, err)
	}

	maxLen := len(curParts)
	if len(latParts) > maxLen {
		maxLen = len(latParts)
	}

	for i := 0; i < maxLen; i++ {
		cv := 0
		if i < len(curParts) {
			cv = curParts[i]
		}
		lv := 0
		if i < len(latParts) {
			lv = latParts[i]
		}
		if lv > cv {
			return true, nil
		}
		if lv < cv {
			return false, nil
		}
	}
	return false, nil // equal
}

// splitVersion parses "2.0.7.17" into [2, 0, 7, 17].
func splitVersion(v string) ([]int, error) {
	parts := strings.Split(v, ".")
	out := make([]int, len(parts))
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("segment %q is not numeric: %w", p, err)
		}
		out[i] = n
	}
	return out, nil
}
