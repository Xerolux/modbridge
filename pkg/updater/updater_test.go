// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package updater

import "testing"

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		name    string
		current string
		latest  string
		want    bool
		wantErr bool
	}{
		{"newer patch", "2.0.7.17", "2.0.7.18", true, false},
		{"equal versions", "2.0.7.17", "2.0.7.17", false, false},
		{"older latest", "2.0.7.18", "2.0.7.17", false, false},
		{"newer minor", "2.0.7.17", "2.1.0.0", true, false},
		{"newer major", "1.9.9.9", "2.0.0.0", true, false},
		{"with v prefix", "2.0.7.17", "v2.0.7.18", true, false},
		{"both v prefix", "v2.0.7.17", "v2.0.7.18", true, false},
		{"different segment count newer", "2.0.7", "2.0.7.1", true, false},
		{"different segment count older", "2.0.7.1", "2.0.7", false, false},
		{"empty current", "", "2.0.7.18", true, false},
		{"dev current", "dev", "2.0.7.18", true, false},
		{"invalid version", "abc.def", "2.0.7.18", false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := CompareVersions(c.current, c.latest)
			if (err != nil) != c.wantErr {
				t.Fatalf("CompareVersions(%q, %q) error = %v, wantErr %v", c.current, c.latest, err, c.wantErr)
			}
			if got != c.want {
				t.Errorf("CompareVersions(%q, %q) = %v, want %v", c.current, c.latest, got, c.want)
			}
		})
	}
}
