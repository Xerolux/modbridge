// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

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

func TestVerifyChecksum(t *testing.T) {
	// Build a valid checksums.txt for a known asset
	assetContent := []byte("fake-binary-content")
	sum := sha256.Sum256(assetContent)
	sumHex := hex.EncodeToString(sum[:])
	assetName := "modbridge-linux-amd64"

	checksumsValid := []byte(sumHex + "  " + assetName + "\n")
	checksumsExtra := []byte("abc123  other-asset\n" + sumHex + "  " + assetName + "\n")
	checksumsMissing := []byte("abcdef  some-other-asset\n")
	checksumsMalformed := []byte("not-a-checksum-line\n")

	cases := []struct {
		name        string
		asset       []byte
		checksums   []byte
		wantErr     error
		wantErrType string // "" or "missing"/"mismatch"
	}{
		{"valid", assetContent, checksumsValid, nil, ""},
		{"valid with extra lines", assetContent, checksumsExtra, nil, ""},
		{"asset not in checksums", assetContent, checksumsMissing, ErrChecksumMissing, "missing"},
		{"checksum mismatch", []byte("tampered"), checksumsValid, ErrChecksumMismatch, "mismatch"},
		{"empty checksums file", assetContent, []byte(""), ErrChecksumMissing, "missing"},
		{"malformed line skipped", assetContent, append(checksumsMalformed, checksumsValid...), nil, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := VerifyChecksum(c.asset, assetName, c.checksums)
			if c.wantErrType == "" && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if c.wantErrType == "missing" && err != ErrChecksumMissing {
				t.Fatalf("expected ErrChecksumMissing, got: %v", err)
			}
			if c.wantErrType == "mismatch" && err != ErrChecksumMismatch {
				t.Fatalf("expected ErrChecksumMismatch, got: %v", err)
			}
		})
	}
}
