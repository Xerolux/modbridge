// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package updater

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
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

func TestSwapBinary(t *testing.T) {
	// Create a temp directory simulating the install dir
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "modbridge")

	// Write the "current" binary
	if err := os.WriteFile(binaryPath, []byte("OLD-BINARY"), 0755); err != nil {
		t.Fatal(err)
	}

	// Write the "new" binary in the SAME directory (required for atomic rename)
	newPath := filepath.Join(dir, "modbridge.new")
	if err := os.WriteFile(newPath, []byte("NEW-BINARY"), 0644); err != nil {
		t.Fatal(err)
	}

	backupPath, err := SwapBinary(newPath, binaryPath)
	if err != nil {
		t.Fatalf("SwapBinary failed: %v", err)
	}

	// New binary should now be at binaryPath
	got, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatalf("reading swapped binary: %v", err)
	}
	if string(got) != "NEW-BINARY" {
		t.Errorf("after swap, binary = %q, want NEW-BINARY", string(got))
	}

	// New binary must be executable (execute bit is not representable on
	// Windows through Go's POSIX-ish file-mode API, so skip the check there).
	info, _ := os.Stat(binaryPath)
	if runtime.GOOS != "windows" && info.Mode()&0100 == 0 {
		t.Error("swapped binary is not executable")
	}

	// Backup should contain the old content
	backup, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("reading backup: %v", err)
	}
	if string(backup) != "OLD-BINARY" {
		t.Errorf("backup = %q, want OLD-BINARY", string(backup))
	}

	// Backup path should be alongside the binary
	if filepath.Dir(backupPath) != dir {
		t.Errorf("backup in wrong dir: %s", filepath.Dir(backupPath))
	}
}

func TestSwapBinary_Rollback(t *testing.T) {
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "modbridge")
	os.WriteFile(binaryPath, []byte("OLD"), 0755)

	// Manually create a backup to simulate pre-swap state
	backupPath := filepath.Join(dir, "modbridge.bak.test")
	os.WriteFile(backupPath, []byte("BACKUP-CONTENT"), 0755)
	// Corrupt the current binary
	os.WriteFile(binaryPath, []byte("CORRUPTED"), 0755)

	if err := RollbackBinary(backupPath); err != nil {
		t.Fatalf("RollbackBinary failed: %v", err)
	}

	// The binary should be restored from backup. Rollback restores by
	// copying backup -> binary (backup is preserved for forensics).
	got, _ := os.ReadFile(binaryPath)
	if string(got) != "BACKUP-CONTENT" {
		t.Errorf("after rollback, binary = %q, want BACKUP-CONTENT", string(got))
	}
}

func TestCheckForUpdate_Cache(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"tag_name":"v2.0.7.18","prerelease":false,"assets":[{"name":"modbridge-linux-amd64","browser_download_url":"http://example.com/a","size":100}]}`))
	}))
	defer srv.Close()

	u := New("Xerolux/modbridge", BuildInfo{Version: "2.0.7.17", OS: "linux", Arch: "amd64"})
	u.apiBase = srv.URL // inject test server

	_, err := u.CheckForUpdate(context.Background())
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	_, err = u.CheckForUpdate(context.Background())
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if callCount != 1 {
		t.Errorf("GitHub API called %d times, expected 1 (cache hit on second call)", callCount)
	}
}

func TestGetStatus_InitialState(t *testing.T) {
	u := New("Xerolux/modbridge", BuildInfo{Version: "2.0.7.17"})
	st := u.GetStatus()
	if st.State != StateIdle {
		t.Errorf("initial state = %v, want %v", st.State, StateIdle)
	}
}

func TestPerformUpdate_RejectsWhenAlreadyRunning(t *testing.T) {
	u := New("Xerolux/modbridge", BuildInfo{Version: "2.0.7.17"})
	u.mu.Lock()
	u.status.State = StateDownloading
	u.mu.Unlock()

	err := u.PerformUpdate(context.Background())
	if err == nil {
		t.Error("expected error when update already in progress")
	}
}

// TestDownloadFile_EmitsProgress verifies that downloadFile reports
// intermediate progress (Fix 4): with a known body size, the updater's status
// must advance beyond the initial 10% before reaching the "complete" mark.
func TestDownloadFile_EmitsProgress(t *testing.T) {
	// Serve a ~1 MB body so progress throttling (≥3% or ≥400ms) is guaranteed
	// to emit at least one intermediate update.
	const bodySize = 1 << 20
	body := bytes.Repeat([]byte("x"), bodySize)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(bodySize))
		w.Write(body)
	}))
	defer srv.Close()

	u := New("Xerolux/modbridge", BuildInfo{Version: "2.0.7.17"})
	dest := filepath.Join(t.TempDir(), "blob")
	if err := downloadFile(context.Background(), u.client, srv.URL, dest, bodySize, u); err != nil {
		t.Fatalf("downloadFile failed: %v", err)
	}

	// Drain the final status. Since progressWriter updates happen during the
	// io.Copy, the last seen progress should have advanced past the 10% start
	// (and at least one update should have fired given 1MB >> 3% threshold).
	st := u.GetStatus()
	if st.Progress <= 10 {
		t.Errorf("expected progress to advance beyond 10 during download, got %d", st.Progress)
	}
	if st.Progress > 55 {
		t.Errorf("progress %d exceeds the download band ceiling 55", st.Progress)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("reading downloaded file: %v", err)
	}
	if len(got) != bodySize {
		t.Errorf("downloaded %d bytes, want %d", len(got), bodySize)
	}
}

// TestDownloadFile_UnknownSizeDoesNotPanic ensures that when neither
// Content-Length nor expectedSize is available, downloadFile still succeeds
// without dividing by zero in progressWriter.
func TestDownloadFile_UnknownSize(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Deliberately no Content-Length; chunked encoding.
		w.Write([]byte("small-body"))
	}))
	defer srv.Close()

	u := New("Xerolux/modbridge", BuildInfo{Version: "2.0.7.17"})
	dest := filepath.Join(t.TempDir(), "blob")
	if err := downloadFile(context.Background(), u.client, srv.URL, dest, 0, u); err != nil {
		t.Fatalf("downloadFile with unknown size failed: %v", err)
	}
	got, _ := os.ReadFile(dest)
	if string(got) != "small-body" {
		t.Errorf("got %q, want small-body", string(got))
	}
}

// TestCheckForUpdate_DoesNotClobberActiveState verifies the state-machine guard
// (Fix 6): when CheckForUpdate runs while another phase is already in progress,
// its deferred Idle reset must not overwrite the active state.
func TestCheckForUpdate_DoesNotClobberActiveState(t *testing.T) {
	// A working GitHub mock so the check itself succeeds.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"tag_name":"v9.9.9.9","prerelease":false,"assets":[]}`))
	}))
	defer srv.Close()

	u := New("Xerolux/modbridge", BuildInfo{Version: "2.0.7.17"})
	u.SetAPIBase(srv.URL)

	// Simulate an update already in flight (runUpdate would have set this).
	u.mu.Lock()
	u.status.State = StateDownloading
	u.status.Progress = 25
	u.mu.Unlock()

	if _, err := u.CheckForUpdate(context.Background()); err != nil {
		t.Fatalf("CheckForUpdate failed: %v", err)
	}

	st := u.GetStatus()
	if st.State == StateIdle {
		t.Errorf("CheckForUpdate clobbered an active %v state back to Idle", StateDownloading)
	}
}

// TestRollbackTargetPath_MultipleBakSegments ensures rollbackTargetPath strips
// at the LAST .bak (Fix 7), so binary names containing ".bak" earlier are
// handled correctly.
func TestRollbackTargetPath_MultipleBakSegments(t *testing.T) {
	cases := []struct {
		name    string
		backup  string
		want    string
		wantErr bool
	}{
		{"standard", filepath.Join("opt", "modbridge.bak.20260711_120000"), filepath.Join("opt", "modbridge"), false},
		{"multiple bak segments", filepath.Join("opt", "modbridge.bak-tool.bak.20260711_120000"), filepath.Join("opt", "modbridge.bak-tool"), false},
		{"no bak segment", filepath.Join("opt", "modbridge.20260711"), "", true},
		{"bak at start (invalid)", filepath.Join("opt", ".bak.something"), "", true}, // i>0 guard rejects leading .bak
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := rollbackTargetPath(c.backup)
			if (err != nil) != c.wantErr {
				t.Fatalf("rollbackTargetPath(%q) err = %v, wantErr %v", c.backup, err, c.wantErr)
			}
			if !c.wantErr && got != c.want {
				t.Errorf("rollbackTargetPath(%q) = %q, want %q", c.backup, got, c.want)
			}
		})
	}
}
