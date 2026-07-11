// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
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

func TestSelectAsset(t *testing.T) {
	assets := []Asset{
		{Name: "modbridge-linux-amd64"},
		{Name: "modbridge-linux-arm64"},
		{Name: "modbridge-linux-arm"},
		{Name: "modbridge-darwin-amd64"},
		{Name: "checksums.txt"},
	}
	cases := []struct {
		name    string
		goos    string
		goarch  string
		want    string
		wantErr bool
	}{
		{"linux amd64", "linux", "amd64", "modbridge-linux-amd64", false},
		{"linux arm64", "linux", "arm64", "modbridge-linux-arm64", false},
		{"linux arm", "linux", "arm", "modbridge-linux-arm", false},
		{"darwin amd64", "darwin", "amd64", "modbridge-darwin-amd64", false},
		{"windows unsupported", "windows", "amd64", "", true},
		{"freebsd unsupported", "freebsd", "amd64", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := SelectAsset(assets, c.goos, c.goarch)
			if (err != nil) != c.wantErr {
				t.Fatalf("SelectAsset err = %v, wantErr %v", err, c.wantErr)
			}
			if !c.wantErr && got.Name != c.want {
				t.Errorf("SelectAsset = %q, want %q", got.Name, c.want)
			}
		})
	}
}

func TestFetchLatestRelease(t *testing.T) {
	// Mock GitHub API response
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/vnd.github+json" {
			t.Errorf("missing Accept header, got %q", r.Header.Get("Accept"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"tag_name": "v2.0.7.18",
			"name": "Release 2.0.7.18",
			"body": "## Changes\n- fix: dashboard widgets\n- feat: update module",
			"html_url": "https://github.com/Xerolux/modbridge/releases/tag/v2.0.7.18",
			"published_at": "2026-07-15T10:00:00Z",
			"prerelease": false,
			"assets": [
				{"name": "modbridge-linux-amd64", "browser_download_url": "https://example.com/modbridge-linux-amd64", "size": 12861136},
				{"name": "checksums.txt", "browser_download_url": "https://example.com/checksums.txt", "size": 200}
			]
		}`))
	}))
	defer srv.Close()

	release, err := fetchLatestRelease(context.Background(), srv.Client(), srv.URL+"/repos/Xerolux/modbridge/releases/latest")
	if err != nil {
		t.Fatalf("fetchLatestRelease failed: %v", err)
	}
	if release.TagName != "v2.0.7.18" {
		t.Errorf("TagName = %q, want v2.0.7.18", release.TagName)
	}
	if release.Version() != "2.0.7.18" {
		t.Errorf("Version() = %q, want 2.0.7.18", release.Version())
	}
	if release.Prerelease {
		t.Error("Prerelease should be false")
	}
	if len(release.Assets) != 2 {
		t.Fatalf("Assets count = %d, want 2", len(release.Assets))
	}
	if release.Assets[0].Name != "modbridge-linux-amd64" {
		t.Errorf("Asset[0].Name = %q", release.Assets[0].Name)
	}
}
