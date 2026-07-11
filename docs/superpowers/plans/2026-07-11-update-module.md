# Update-Module Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a professional, fully-automatic update module to ModBridge that checks GitHub Releases, downloads the matching binary for the current architecture, verifies its SHA256 checksum, and atomically swaps the running binary — triggered entirely from the WebUI.

**Architecture:** New `pkg/updater/` Go package handles GitHub-API calls, checksum verification, and atomic binary swap. Three new admin-only API endpoints in `pkg/api/` expose check/perform/status. The frontend gets a new "Update" section on the existing `/system` page. Reuses the existing `restartSignal` channel pattern so `main.go` performs its normal graceful shutdown and systemd restarts the new binary.

**Tech Stack:** Go 1.26 (stdlib `net/http`, `crypto/sha256`, `os`, `runtime`), Vue 3 Composition API, PrimeVue components, vue-i18n. No new Go dependencies — everything via stdlib.

## Global Constraints

- Go 1.26.4, CGO_ENABLED=1 (for go-sqlite3, unaffected by this feature)
- All Go files use the copyright header:
  ```
  // Copyright (c) 2026 Xerolux. All rights reserved.
  // ModBridge — Modbus TCP Proxy Manager
  // Created by Xerolux
  // https://github.com/Xerolux/modbridge
  ```
- Package names: lowercase single word (`updater`)
- Exported symbols: `CamelCase`; unexported: `camelCase`; receiver: short type abbreviation (`u *Updater`)
- Receiver for Server methods: `s *Server` (existing convention)
- Errors wrapped with `%w`; doc comments on all exported types/functions
- Frontend uses `<script setup>`, vue-i18n for all user-facing strings, PrimeVue components
- Update endpoints are admin-only via `requirePermission(rbac.PermSystemRestart)`
- No new external Go dependencies (stdlib only)
- Version injected via ldflags: `-X main.Version=... -X main.BuildTime=...`

---

## File Structure

**New files:**
- `pkg/updater/updater.go` — Core `Updater` type, status state machine, `CheckForUpdate`, `PerformUpdate`, `GetStatus`
- `pkg/updater/github.go` — GitHub releases API client, asset selection per platform
- `pkg/updater/verify.go` — SHA256 checksum parsing and verification
- `pkg/updater/swap.go` — Atomic binary swap with backup and rollback
- `pkg/updater/version.go` — Semantic version comparison
- `pkg/updater/updater_test.go` — All unit tests (table-driven)
- `pkg/api/handlers_update.go` — Three HTTP handlers for check/perform/status

**Modified files:**
- `pkg/api/server.go` — Add `updater *updater.Updater` field, register 3 routes, wire updater in constructor
- `frontend/src/views/SystemInfo.vue` — Add "Update" section with version comparison, changelog, progress
- `frontend/src/i18n.js` — Add `update` block (de + en)

**Not modified:** `main.go` (restartSignal pattern already exists and is reused as-is)

---

## Task 1: Version comparison logic

**Files:**
- Create: `pkg/updater/version.go`
- Test: `pkg/updater/updater_test.go`

**Interfaces:**
- Produces: `CompareVersions(current, latest string) (bool, error)` — returns true if `latest > current`. Used by Task 6 (CheckForUpdate).

- [ ] **Step 1: Write the failing test**

Create `pkg/updater/updater_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/updater/ -run TestCompareVersions -v`
Expected: FAIL — package `updater` not found / `CompareVersions` undefined.

- [ ] **Step 3: Write minimal implementation**

Create `pkg/updater/version.go`:

```go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/updater/ -run TestCompareVersions -v`
Expected: PASS — all 12 subtests pass.

- [ ] **Step 5: Commit**

```bash
git add pkg/updater/version.go pkg/updater/updater_test.go
git commit -m "feat(updater): add version comparison logic"
```

---

## Task 2: SHA256 checksum verification

**Files:**
- Create: `pkg/updater/verify.go`
- Modify: `pkg/updater/updater_test.go` (append tests)

**Interfaces:**
- Produces: `VerifyChecksum(assetBytes []byte, assetName string, checksumsTxt []byte) error`. Used by Task 6 (PerformUpdate).
- Produces: sentinel errors `ErrChecksumMissing`, `ErrChecksumMismatch`.

- [ ] **Step 1: Write the failing test**

Append to `pkg/updater/updater_test.go`:

```go
import (
	// add to existing import block:
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/updater/ -run TestVerifyChecksum -v`
Expected: FAIL — `VerifyChecksum`, `ErrChecksumMissing`, `ErrChecksumMismatch` undefined.

- [ ] **Step 3: Write minimal implementation**

Create `pkg/updater/verify.go`:

```go
// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// ErrChecksumMissing is returned when the asset name does not appear in
// the downloaded checksums.txt file.
var ErrChecksumMissing = errors.New("asset not found in checksums file")

// ErrChecksumMismatch is returned when the computed SHA256 of the
// downloaded asset does not match the value recorded in checksums.txt.
var ErrChecksumMismatch = errors.New("checksum mismatch — download may be corrupted or tampered")

// VerifyChecksum parses checksumsTxt (standard sha256sum output format:
// "<hex>  <filename>" per line, two spaces) and confirms that the SHA256
// of assetBytes matches the entry for assetName.
func VerifyChecksum(assetBytes []byte, assetName string, checksumsTxt []byte) error {
	expected, err := findChecksum(checksumsTxt, assetName)
	if err != nil {
		return err
	}

	computed := sha256.Sum256(assetBytes)
	computedHex := hex.EncodeToString(computed[:])

	if !strings.EqualFold(computedHex, expected) {
		return fmt.Errorf("%w: expected %s, got %s", ErrChecksumMismatch, expected, computedHex)
	}
	return nil
}

// findChecksum scans the sha256sum-formatted text for the given filename
// and returns the hex digest. Lines that don't match the expected format
// are silently skipped (defensive parsing).
func findChecksum(checksumsTxt []byte, assetName string) (string, error) {
	lines := strings.Split(string(checksumsTxt), "\n")
	for _, line := range lines {
		// sha256sum format: "<64-hex>  <filename>" (two spaces)
		parts := strings.SplitN(strings.TrimSpace(line), "  ", 2)
		if len(parts) != 2 {
			continue
		}
		digest := strings.TrimSpace(parts[0])
		name := strings.TrimSpace(parts[1])
		if name == assetName && len(digest) == 64 {
			return digest, nil
		}
	}
	return "", fmt.Errorf("%w: %s", ErrChecksumMissing, assetName)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/updater/ -run TestVerifyChecksum -v`
Expected: PASS — all 6 subtests pass.

- [ ] **Step 5: Commit**

```bash
git add pkg/updater/verify.go pkg/updater/updater_test.go
git commit -m "feat(updater): add SHA256 checksum verification"
```

---

## Task 3: GitHub releases API client + asset selection

**Files:**
- Create: `pkg/updater/github.go`
- Modify: `pkg/updater/updater_test.go` (append tests)

**Interfaces:**
- Produces: `ReleaseInfo`, `Asset` types.
- Produces: `SelectAsset(assets []Asset, goos, goarch string) (Asset, error)`.
- Produces: `fetchLatestRelease(ctx, client, repo) (*ReleaseInfo, error)` (unexported; tested via a test helper that uses `httptest`).

- [ ] **Step 1: Write the failing test**

Append to `pkg/updater/updater_test.go`:

```go
import (
	// add to existing import block:
	"context"
	"net/http"
	"net/http/httptest"
)

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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/updater/ -run "TestSelectAsset|TestFetchLatestRelease" -v`
Expected: FAIL — types and functions undefined.

- [ ] **Step 3: Write minimal implementation**

Create `pkg/updater/github.go`:

```go
// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ReleaseInfo describes a GitHub release relevant to the update flow.
type ReleaseInfo struct {
	TagName      string    `json:"tag_name"`
	Name         string    `json:"name"`
	Body         string    `json:"body"` // release notes / changelog
	HTMLURL      string    `json:"html_url"`
	PublishedAt  time.Time `json:"published_at"`
	Prerelease   bool      `json:"prerelease"`
	Assets       []Asset   `json:"assets"`
}

// Version returns the tag name without the leading "v" prefix.
func (r *ReleaseInfo) Version() string {
	return strings.TrimPrefix(r.TagName, "v")
}

// Asset is a single downloadable file attached to a release.
type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}

// SelectAsset picks the binary asset matching the given GOOS/GOARCH.
// Non-binary assets (checksums.txt, etc.) are ignored. Returns an error
// if no matching asset exists for the platform.
func SelectAsset(assets []Asset, goos, goarch string) (Asset, error) {
	want := expectedAssetName(goos, goarch)
	if want == "" {
		return Asset{}, fmt.Errorf("updates are not supported on %s/%s", goos, goarch)
	}
	for _, a := range assets {
		if a.Name == want {
			return a, nil
		}
	}
	return Asset{}, fmt.Errorf("no release asset %q found for %s/%s", want, goos, goarch)
}

// expectedAssetName maps runtime.GOOS/GOARCH to the asset naming used by
// the release workflow (.github/workflows/release.yml). Returns empty
// string for unsupported platforms.
func expectedAssetName(goos, goarch string) string {
	switch goos {
	case "linux":
		switch goarch {
		case "amd64":
			return "modbridge-linux-amd64"
		case "arm64":
			return "modbridge-linux-arm64"
		case "arm":
			return "modbridge-linux-arm"
		}
	case "darwin":
		switch goarch {
		case "amd64":
			return "modbridge-darwin-amd64"
		case "arm64":
			return "modbridge-darwin-arm64"
		}
	}
	return ""
}

// githubBaseURL is the API root. Overridable in tests via fetchLatestRelease's
// URL argument; in production this is always "https://api.github.com".
const githubBaseURL = "https://api.github.com"

// fetchLatestRelease queries the GitHub API for the latest release of repo
// (format "owner/name"). The apiURL argument allows tests to point at an
// httptest server.
func fetchLatestRelease(ctx context.Context, client *http.Client, apiURL string) (*ReleaseInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building GitHub API request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "ModBridge-Updater")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decoding GitHub API response: %w", err)
	}
	return &release, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/updater/ -run "TestSelectAsset|TestFetchLatestRelease" -v`
Expected: PASS — all subtests pass.

- [ ] **Step 5: Commit**

```bash
git add pkg/updater/github.go pkg/updater/updater_test.go
git commit -m "feat(updater): add GitHub releases API client + asset selection"
```

---

## Task 4: Atomic binary swap with backup and rollback

**Files:**
- Create: `pkg/updater/swap.go`
- Modify: `pkg/updater/updater_test.go` (append tests)

**Interfaces:**
- Produces: `SwapBinary(newBinaryPath string) (backupPath string, err error)`.
- Produces: `RollbackBinary(backupPath string) error`.

- [ ] **Step 1: Write the failing test**

Append to `pkg/updater/updater_test.go`:

```go
import (
	// add to existing import block:
	"os"
	"path/filepath"
)

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

	backupPath, err := SwapBinary(newPath)
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

	// New binary must be executable
	info, _ := os.Stat(binaryPath)
	if info.Mode()&0100 == 0 {
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
	// copying backup → binary (backup is preserved for forensics).
	got, _ := os.ReadFile(binaryPath)
	if string(got) != "BACKUP-CONTENT" {
		t.Errorf("after rollback, binary = %q, want BACKUP-CONTENT", string(got))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/updater/ -run "TestSwapBinary" -v`
Expected: FAIL — `SwapBinary`, `RollbackBinary` undefined.

- [ ] **Step 3: Write minimal implementation**

Create `pkg/updater/swap.go`:

```go
// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package updater

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SwapBinary replaces the currently running binary (located via os.Executable())
// with the file at newBinaryPath. The old binary is preserved at
// "<binaryPath>.bak.<timestamp>". Returns the backup path.
//
// The new binary is expected to reside in the same directory as the target
// (os.Rename is only atomic within a single filesystem).
func SwapBinary(newBinaryPath string) (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locating current executable: %w", err)
	}

	// Resolve symlinks — /opt/modbridge/modbridge may be a symlink
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return "", fmt.Errorf("resolving executable path: %w", err)
	}

	backupPath := execPath + ".bak." + time.Now().Format("20060102_150405")

	// Step 1: move current binary aside (backup)
	if err := os.Rename(execPath, backupPath); err != nil {
		return "", fmt.Errorf("creating backup %s: %w", backupPath, err)
	}

	// Step 2: move new binary into place
	if err := os.Rename(newBinaryPath, execPath); err != nil {
		// Rollback: restore the old binary
		if rbErr := os.Rename(backupPath, execPath); rbErr != nil {
			return "", fmt.Errorf("swap failed (%v) and rollback also failed (%v) — binary at %s, manual recovery needed", err, rbErr, backupPath)
		}
		return "", fmt.Errorf("installing new binary: %w (rolled back)", err)
	}

	// Step 3: ensure executable bit
	if err := os.Chmod(execPath, 0755); err != nil {
		return backupPath, fmt.Errorf("setting executable permissions: %w", err)
	}

	return backupPath, nil
}

// RollbackBinary copies the backup file back over the current binary path.
// The backup file itself is preserved for forensics. Used when the new
// binary is detected as broken after a restart (manual or future auto-detect).
func RollbackBinary(backupPath string) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locating current executable: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("resolving executable path: %w", err)
	}

	// Copy backup → exec (don't rename, keep backup for forensics)
	src, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("opening backup: %w", err)
	}
	defer src.Close()

	dst, err := os.OpenFile(execPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("opening target for rollback: %w", err)
	}
	defer dst.Close()

	if _, err := copyAll(dst, src); err != nil {
		return fmt.Errorf("copying backup to target: %w", err)
	}
	return nil
}

// copyAll is a small helper around io.Copy for the rollback path.
func copyAll(dst *os.File, src *os.File) (int64, error) {
	buf := make([]byte, 32*1024)
	var total int64
	for {
		n, err := src.Read(buf)
		if n > 0 {
			written, werr := dst.Write(buf[:n])
			total += int64(written)
			if werr != nil {
				return total, werr
			}
		}
		if err != nil {
			if err.Error() == "EOF" {
				return total, nil
			}
			return total, err
		}
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/updater/ -run "TestSwapBinary" -v`
Expected: PASS — both subtests pass.

- [ ] **Step 5: Commit**

```bash
git add pkg/updater/swap.go pkg/updater/updater_test.go
git commit -m "feat(updater): add atomic binary swap with backup and rollback"
```

---

## Task 5: Updater core — CheckForUpdate, PerformUpdate, status state machine

**Files:**
- Create: `pkg/updater/updater.go`
- Modify: `pkg/updater/updater_test.go` (append tests)

**Interfaces:**
- Produces: `Updater` type, `BuildInfo`, `UpdateStatus`, `State` constants.
- Produces: `New(repo string, current BuildInfo) *Updater`.
- Produces: `(*Updater).CheckForUpdate(ctx) (*ReleaseInfo, error)` — cached 60s.
- Produces: `(*Updater).PerformUpdate(ctx) error` — async, updates status.
- Produces: `(*Updater).GetStatus() UpdateStatus`.

- [ ] **Step 1: Write the failing test**

Append to `pkg/updater/updater_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/updater/ -run "TestCheckForUpdate_Cache|TestGetStatus_InitialState|TestPerformUpdate_RejectsWhenAlreadyRunning" -v`
Expected: FAIL — `Updater`, `New`, `BuildInfo`, state constants undefined.

- [ ] **Step 3: Write minimal implementation**

Create `pkg/updater/updater.go`:

```go
// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package updater

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// State represents the current phase of an update operation.
type State string

const (
	StateIdle        State = "idle"
	StateChecking    State = "checking"
	StateDownloading State = "downloading"
	StateVerifying   State = "verifying"
	StateSwapping    State = "swapping"
	StateRestarting  State = "restarting"
	StateDone        State = "done"
	StateError       State = "error"
)

// BuildInfo describes the currently running binary.
type BuildInfo struct {
	Version   string
	BuildTime string
	GoVersion string
	OS        string
	Arch      string
}

// UpdateStatus is the pollable progress of an update operation.
type UpdateStatus struct {
	State     State     `json:"state"`
	Progress  int       `json:"progress"`
	Message   string    `json:"message"`
	Error     string    `json:"error,omitempty"`
	StartedAt time.Time `json:"started_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ErrUpdateInProgress is returned when PerformUpdate is called while an
// update is already running.
var ErrUpdateInProgress = errors.New("an update is already in progress")

// Updater coordinates GitHub release checks and atomic binary updates.
type Updater struct {
	repo    string
	current BuildInfo
	client  *http.Client

	apiBase string // overridable in tests; default githubBaseURL

	mu            sync.RWMutex
	status        UpdateStatus
	cachedRelease *ReleaseInfo
	cacheExpiry   time.Time
}

// New creates an Updater for the given repo (e.g. "Xerolux/modbridge").
func New(repo string, current BuildInfo) *Updater {
	return &Updater{
		repo:    repo,
		current: current,
		client:  &http.Client{Timeout: 30 * time.Second},
		apiBase: githubBaseURL,
		status: UpdateStatus{
			State:     StateIdle,
			UpdatedAt: time.Now(),
		},
	}
}

// GetStatus returns a snapshot of the current update progress. Safe for
// concurrent use.
func (u *Updater) GetStatus() UpdateStatus {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.status
}

func (u *Updater) setStatus(state State, progress int, message string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.status.State = state
	u.status.Progress = progress
	u.status.Message = message
	u.status.UpdatedAt = time.Now()
	if state == StateDownloading && u.status.StartedAt.IsZero() {
		u.status.StartedAt = time.Now()
	}
	if state == StateIdle || state == StateDone {
		u.status.StartedAt = time.Time{}
	}
}

func (u *Updater) setError(message string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.status.State = StateError
	u.status.Message = message
	u.status.Error = message
	u.status.UpdatedAt = time.Now()
}

// CheckForUpdate queries GitHub for the latest release. Results are cached
// for 60 seconds to respect API rate limits.
func (u *Updater) CheckForUpdate(ctx context.Context) (*ReleaseInfo, error) {
	u.mu.RLock()
	if u.cachedRelease != nil && time.Now().Before(u.cacheExpiry) {
		r := u.cachedRelease
		u.mu.RUnlock()
		return r, nil
	}
	u.mu.RUnlock()

	u.setStatus(StateChecking, 0, "checking")
	defer u.setStatus(StateIdle, 0, "")

	apiURL := fmt.Sprintf("%s/repos/%s/releases/latest", u.apiBase, u.repo)
	release, err := fetchLatestRelease(ctx, u.client, apiURL)
	if err != nil {
		return nil, err
	}

	u.mu.Lock()
	u.cachedRelease = release
	u.cacheExpiry = time.Now().Add(60 * time.Second)
	u.mu.Unlock()

	return release, nil
}

// PerformUpdate runs the full update flow asynchronously: download the
// binary, verify its checksum, swap it in atomically, then signal restart.
// Progress is reported via GetStatus(). Returns ErrUpdateInProgress if an
// update is already running.
func (u *Updater) PerformUpdate(ctx context.Context) error {
	u.mu.RLock()
	running := u.status.State != StateIdle && u.status.State != StateDone && u.status.State != StateError
	u.mu.RUnlock()
	if running {
		return ErrUpdateInProgress
	}

	release, err := u.CheckForUpdate(ctx)
	if err != nil {
		return fmt.Errorf("checking for update: %w", err)
	}

	asset, err := SelectAsset(release.Assets, u.current.OS, u.current.Arch)
	if err != nil {
		return fmt.Errorf("selecting asset: %w", err)
	}

	// Find checksums.txt asset
	var checksumsAsset *Asset
	for i := range release.Assets {
		if release.Assets[i].Name == "checksums.txt" {
			checksumsAsset = &release.Assets[i]
			break
		}
	}
	if checksumsAsset == nil {
		return errors.New("checksums.txt not found in release assets — cannot verify integrity")
	}

	go u.runUpdate(ctx, release, asset, checksumsAsset)
	return nil
}

// runUpdate executes the update phases. Runs in its own goroutine.
func (u *Updater) runUpdate(ctx context.Context, release *ReleaseInfo, asset Asset, checksumsAsset *Asset) {
	// Phase 1: Download binary
	u.setStatus(StateDownloading, 10, "downloading "+asset.Name)
	execDir := executableDir()
	if execDir == "" {
		u.setError("could not determine executable directory for temp file")
		return
	}
	tempPath := filepath.Join(execDir, ".modbridge.update.tmp")
	if err := downloadFile(ctx, u.client, asset.URL, tempPath, asset.Size, u); err != nil {
		u.setError("download failed: " + err.Error())
		os.Remove(tempPath)
		return
	}
	u.setStatus(StateDownloading, 60, "download complete")

	// Phase 2: Download checksums and verify
	u.setStatus(StateVerifying, 70, "verifying checksum")
	checksumsBytes, err := downloadBytes(ctx, u.client, checksumsAsset.URL)
	if err != nil {
		u.setError("downloading checksums.txt: " + err.Error())
		os.Remove(tempPath)
		return
	}
	assetBytes, err := os.ReadFile(tempPath)
	if err != nil {
		u.setError("reading downloaded binary: " + err.Error())
		os.Remove(tempPath)
		return
	}
	if err := VerifyChecksum(assetBytes, asset.Name, checksumsBytes); err != nil {
		u.setError("verification failed: " + err.Error())
		os.Remove(tempPath)
		return
	}
	u.setStatus(StateVerifying, 85, "checksum verified")

	// Phase 3: Atomic swap
	u.setStatus(StateSwapping, 90, "swapping binary")
	if _, err := SwapBinary(tempPath); err != nil {
		u.setError("swap failed: " + err.Error())
		os.Remove(tempPath)
		return
	}

	// Phase 4: Signal restart (caller closes restartSignal channel)
	u.setStatus(StateRestarting, 95, "restarting service")
	// The API handler triggers the actual restart via the Server's
	// restartSignal after observing StateRestarting. This separation keeps
	// the updater free of HTTP-server coupling.
	u.setStatus(StateDone, 100, "update installed — restart pending")
}

// executableDir returns the directory of the running binary, or "" on error.
func executableDir() string {
	p, err := os.Executable()
	if err != nil {
		return ""
	}
	p, err = filepath.EvalSymlinks(p)
	if err != nil {
		return ""
	}
	return filepath.Dir(p)
}

// downloadFile fetches url into destPath, reporting progress via the updater.
func downloadFile(ctx context.Context, client *http.Client, url, destPath string, expectedSize int64, u *Updater) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

// downloadBytes fetches url and returns the full body.
func downloadBytes(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/updater/ -v`
Expected: PASS — all tests (Tasks 1-5) pass.

- [ ] **Step 5: Commit**

```bash
git add pkg/updater/updater.go pkg/updater/updater_test.go
git commit -m "feat(updater): add Updater core with check/perform/status state machine"
```

---

## Task 6: Wire updater into API server + three endpoints

**Files:**
- Create: `pkg/api/handlers_update.go`
- Modify: `pkg/api/server.go` — add `updater` field, register routes, construct updater in `New`/constructor

**Interfaces:**
- Consumes: `*updater.Updater`, `updater.BuildInfo`, `updater.UpdateStatus`, `updater.ReleaseInfo`, `updater.New`, `u.CheckForUpdate`, `u.PerformUpdate`, `u.GetStatus` (from Task 5).
- Produces: HTTP handlers `handleUpdateCheck`, `handleUpdatePerform`, `handleUpdateStatus` registered at `/api/update/check`, `/api/update/perform`, `/api/update/status`.

- [ ] **Step 1: Inspect current Server struct and constructor**

Read `pkg/api/server.go:97-200` to find:
- The `Server` struct fields (line ~98)
- The constructor function (search for `func New` — likely near line 130-180)
- The route registration in `Routes()` method (line ~200-280)
- How `restartSignal` is initialized (line 176: `make(chan struct{})`)
- How `requirePermission` is used (line 59)

- [ ] **Step 2: Add updater field and wiring to Server**

Modify `pkg/api/server.go`:

The current constructor signature is `func NewServer(cfg *config.Manager, mgr *manager.Manager, a *auth.Authenticator, l *logger.Logger, db *database.DB) *Server` (line 133). It does NOT take version/buildTime. Add two parameters:

Change the signature to:
```go
func NewServer(cfg *config.Manager, mgr *manager.Manager, a *auth.Authenticator, l *logger.Logger, db *database.DB, version, buildTime string) *Server {
```

In the `Server` struct (after line 112, after `auditor *audit.Auditor`), add:
```go
	updater *updater.Updater
```

Add these imports to the import block:
```go
	"runtime"

	"modbridge/pkg/updater"
```

In the constructor's return struct literal (after `restartSignal: make(chan struct{}),` at line 176), add:
```go
		updater: updater.New("Xerolux/modbridge", updater.BuildInfo{
			Version:   version,
			BuildTime: buildTime,
			GoVersion: runtime.Version(),
			OS:        runtime.GOOS,
			Arch:      runtime.GOARCH,
		}),
```

Update the call site in `main.go:123` from:
```go
	apiServer := api.NewServer(cfgMgr, mgr, authenticator, l, db)
```
to:
```go
	apiServer := api.NewServer(cfgMgr, mgr, authenticator, l, db, Version, BuildTime)
```

`Version` and `BuildTime` are main.go's package-level variables (main.go:27-29).

- [ ] **Step 3: Register the three routes**

In the `Routes()` method, `authMW` and `csrfMW` are **local variables** (line 231-232), not methods. They are composed middleware wrappers (`compose(...)` returns `func(http.HandlerFunc) http.HandlerFunc`). Follow the exact same pattern as the existing `/api/system/*` routes (lines 275-280): wrap with `authMW` for read endpoints, `csrfMW` for state-changing endpoints.

Add these three lines after line 280 (after `/api/system/diagnostics/connectivity`):
```go
	// Update endpoints — admin-only (RBAC checked inside handlers via requirePermission)
	mux.HandleFunc("/api/update/check", authMW(s.handleUpdateCheck))
	mux.HandleFunc("/api/update/perform", csrfMW(s.handleUpdatePerform))
	mux.HandleFunc("/api/update/status", authMW(s.handleUpdateStatus))
```

**RBAC is enforced inside the handlers**, not as middleware. This follows the existing pattern: `handleSystemRestart`, `handleUserByID`, etc. all call `s.requirePermission(w, r, rbac.Perm...)` as their first line. Each of the three new handlers must begin with:
```go
	if s.requirePermission(w, r, rbac.PermSystemRestart) == nil {
		return
	}
```
`requirePermission` (server.go:59) writes the 401/403 response itself and returns nil on failure. This is already shown in the handler code in Step 4 below.

- [ ] **Step 4: Create the handlers**

Create `pkg/api/handlers_update.go`:

```go
// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package api

import (
	"net/http"
	"time"

	"modbridge/pkg/rbac"
	"modbridge/pkg/updater"
)

// updateCheckResponse is returned by GET /api/update/check.
type updateCheckResponse struct {
	CurrentVersion   string `json:"current_version"`
	CurrentBuildTime string `json:"current_build_time"`
	GoVersion        string `json:"go_version"`
	OS               string `json:"os"`
	Arch             string `json:"arch"`
	LatestVersion    string `json:"latest_version"`
	UpdateAvailable  bool   `json:"update_available"`
	AssetUnavailable bool   `json:"asset_unavailable,omitempty"`
	ReleaseNotes     string `json:"release_notes,omitempty"`
	ReleaseURL       string `json:"release_url,omitempty"`
	PublishedAt      string `json:"published_at,omitempty"`
	Prerelease       bool   `json:"prerelease"`
	AssetName        string `json:"asset_name,omitempty"`
	AssetSize        int64  `json:"asset_size,omitempty"`
}

// handleUpdateCheck queries GitHub for the latest release and reports
// whether an update is available.
func (s *Server) handleUpdateCheck(w http.ResponseWriter, r *http.Request) {
	if s.requirePermission(w, r, rbac.PermSystemRestart) == nil {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	release, err := s.updater.CheckForUpdate(r.Context())
	resp := updateCheckResponse{
		CurrentVersion: s.updater.CurrentVersion(),
		CurrentBuildTime: s.updater.BuildTime(),
		GoVersion: s.updater.GoVersion(),
		OS: s.updater.OS(),
		Arch: s.updater.Arch(),
	}
	if err != nil {
		resp.UpdateAvailable = false
		w.Header().Set("Content-Type", "application/json")
		s.writeJSON(w, resp)
		return
	}

	resp.LatestVersion = release.Version()
	resp.ReleaseNotes = release.Body
	resp.ReleaseURL = release.HTMLURL
	resp.PublishedAt = release.PublishedAt.Format("2006-01-02T15:04:05Z")
	resp.Prerelease = release.Prerelease

	newer, _ := updater.CompareVersions(resp.CurrentVersion, resp.LatestVersion)
	resp.UpdateAvailable = newer

	if newer {
		asset, err := updater.SelectAsset(release.Assets, resp.OS, resp.Arch)
		if err != nil {
			resp.AssetUnavailable = true
		} else {
			resp.AssetName = asset.Name
			resp.AssetSize = asset.Size
		}
	}

	w.Header().Set("Content-Type", "application/json")
	s.writeJSON(w, resp)
}

// handleUpdatePerform starts the update process asynchronously.
func (s *Server) handleUpdatePerform(w http.ResponseWriter, r *http.Request) {
	if s.requirePermission(w, r, rbac.PermSystemRestart) == nil {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := s.updater.PerformUpdate(r.Context()); err != nil {
		status := http.StatusInternalServerError
		if err == updater.ErrUpdateInProgress {
			status = http.StatusConflict
		}
		http.Error(w, err.Error(), status)
		return
	}

	// Observe state: when updater reaches StateDone, trigger restart.
	go func() {
		// Poll briefly for the done state, then signal restart.
		// The updater's runUpdate goroutine sets StateDone after swap.
		for i := 0; i < 60; i++ { // max ~60 seconds
			st := s.updater.GetStatus()
			if st.State == updater.StateDone {
				close(s.restartSignal)
				return
			}
			if st.State == updater.StateError {
				return // don't restart on error
			}
			time.Sleep(time.Second)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	s.writeJSON(w, map[string]bool{"job_started": true})
}

// handleUpdateStatus returns the current update progress for polling.
func (s *Server) handleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	if s.requirePermission(w, r, rbac.PermSystemRestart) == nil {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	s.writeJSON(w, s.updater.GetStatus())
}
```

**Important:** The handler references `s.updater.CurrentVersion()`, `.BuildTime()`, `.GoVersion()`, `.OS()`, `.Arch()`. Add these accessor methods to `pkg/updater/updater.go` (after the `New` function):
```go
// CurrentVersion returns the version string of the running binary.
func (u *Updater) CurrentVersion() string { return u.current.Version }

// BuildTime returns the build timestamp of the running binary.
func (u *Updater) BuildTime() string { return u.current.BuildTime }

// GoVersion returns the Go runtime version.
func (u *Updater) GoVersion() string { return u.current.GoVersion }

// OS returns runtime.GOOS of the running binary.
func (u *Updater) OS() string { return u.current.OS }

// Arch returns runtime.GOARCH of the running binary.
func (u *Updater) Arch() string { return u.current.Arch }
```

Fix imports in `handlers_update.go`: needs `"net/http"`, `"time"`, `"modbridge/pkg/rbac"`, `"modbridge/pkg/updater"`. Remove `"encoding/json"` (unused — `writeJSON` handles encoding). The final import block:
```go
import (
	"net/http"
	"time"

	"modbridge/pkg/rbac"
	"modbridge/pkg/updater"
)
```

- [ ] **Step 5: Build and verify compilation**

Run: `go build ./...`
Expected: Builds without errors.

Run: `go vet ./pkg/api/ ./pkg/updater/`
Expected: No issues.

- [ ] **Step 6: Commit**

```bash
git add pkg/api/handlers_update.go pkg/api/server.go pkg/updater/updater.go
git commit -m "feat(api): add /api/update/check, /perform, /status endpoints"
```

---

## Task 7: Frontend i18n keys

**Files:**
- Modify: `frontend/src/i18n.js`

**Interfaces:**
- Produces: `update.*` translation keys used by Task 8.

- [ ] **Step 1: Add the update block to German translations**

In `frontend/src/i18n.js`, after the `system:` block in the German `de` object (around line 155, before the `// Control` comment), insert:

```js
  // Update
  update: {
    title: 'Update',
    installed: 'Installiert',
    latest: 'Neueste Version',
    available: 'Update verfügbar',
    upToDate: 'Aktuell',
    assetUnavailable: 'Kein Binary für diese Plattform',
    checkAgain: 'Erneut prüfen',
    install: 'Update installieren',
    viewOnGithub: 'Auf GitHub ansehen',
    confirmTitle: 'Update installieren?',
    confirmMessage: 'Der Dienst wird für ca. 5 Sekunden neu gestartet. Bestehende Proxy-Verbindungen werden unterbrochen.',
    installSuccess: 'Update erfolgreich. Die Seite wird neu geladen.',
    installFailed: 'Update fehlgeschlagen: {error}',
    checkFailed: 'Update-Check fehlgeschlagen. Internetverbindung prüfen.',
    alreadyRunning: 'Update läuft bereits.',
    state: {
      idle: 'Bereit',
      checking: 'Prüfe…',
      downloading: 'Lade herunter…',
      verifying: 'Verifiziere Prüfsumme…',
      swapping: 'Tausche Binary…',
      restarting: 'Starte neu…',
      done: 'Fertig',
      error: 'Fehler'
    }
  },
```

- [ ] **Step 2: Add the update block to English translations**

After the English `system:` block (around line 507, before the English `// Control` comment), insert the same structure with English values:

```js
  // Update
  update: {
    title: 'Update',
    installed: 'Installed',
    latest: 'Latest Version',
    available: 'Update available',
    upToDate: 'Up to date',
    assetUnavailable: 'No binary for this platform',
    checkAgain: 'Check again',
    install: 'Install update',
    viewOnGithub: 'View on GitHub',
    confirmTitle: 'Install update?',
    confirmMessage: 'The service will restart for about 5 seconds. Existing proxy connections will be interrupted.',
    installSuccess: 'Update successful. Reloading page.',
    installFailed: 'Update failed: {error}',
    checkFailed: 'Update check failed. Check internet connection.',
    alreadyRunning: 'Update already running.',
    state: {
      idle: 'Ready',
      checking: 'Checking…',
      downloading: 'Downloading…',
      verifying: 'Verifying checksum…',
      swapping: 'Swapping binary…',
      restarting: 'Restarting…',
      done: 'Done',
      error: 'Error'
    }
  },
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/i18n.js
git commit -m "feat(i18n): add update module translation keys (de + en)"
```

---

## Task 8: Frontend Update section on System page

**Files:**
- Modify: `frontend/src/views/SystemInfo.vue`

**Interfaces:**
- Consumes: `update.*` i18n keys (Task 7), `/api/update/*` endpoints (Task 6).

- [ ] **Step 1: Read the current SystemInfo.vue structure**

Read `frontend/src/views/SystemInfo.vue` to find:
- Where the `<template>` sections end (to insert the new Update card before `</template>`)
- The `<script setup>` imports and existing reactive state
- How other API calls are made (axios import pattern)

- [ ] **Step 2: Add the Update card to the template**

Before the closing `</template>` tag (but after the last existing section, likely the "Port-Verwaltung" section), insert:

```html
    <!-- ── Update Section ──────────────────────────────────────── -->
    <section class="glass-card rounded-[20px] p-5 mt-6">
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-lg font-bold text-[var(--text-primary)] flex items-center gap-2">
          <i class="pi pi-cloud-download"></i>
          {{ t('update.title') }}
        </h2>
        <Badge
          :severity="updateData.update_available ? (updateData.asset_unavailable ? 'warn' : 'warn') : 'success'"
          :value="updateData.update_available ? t('update.available') : t('update.upToDate')"
        />
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 mb-4">
        <div class="p-3 rounded-xl border border-[var(--border-subtle)] bg-[var(--bg-panel-item)]">
          <div class="text-xs uppercase tracking-wider text-[var(--text-muted)] mb-1">{{ t('update.installed') }}</div>
          <div class="text-lg font-bold text-[var(--text-primary)]">{{ updateData.current_version || '—' }}</div>
          <div class="text-xs text-[var(--text-muted)] mt-1">
            {{ updateData.os }}/{{ updateData.arch }} · {{ updateData.go_version }}
          </div>
        </div>
        <div class="p-3 rounded-xl border border-[var(--border-subtle)] bg-[var(--bg-panel-item)]">
          <div class="text-xs uppercase tracking-wider text-[var(--text-muted)] mb-1">{{ t('update.latest') }}</div>
          <div class="text-lg font-bold text-[var(--text-primary)]">{{ updateData.latest_version || '—' }}</div>
          <div class="text-xs text-[var(--text-muted)] mt-1" v-if="updateData.published_at">
            {{ formatDate(updateData.published_at) }}
          </div>
        </div>
      </div>

      <!-- Changelog -->
      <div v-if="updateData.release_notes" class="mb-4">
        <pre class="text-xs text-[var(--text-secondary)] bg-[var(--bg-panel-item)] p-3 rounded-xl border border-[var(--border-subtle)] whitespace-pre-wrap max-h-48 overflow-y-auto">{{ updateData.release_notes }}</pre>
      </div>

      <!-- Error from check -->
      <div v-if="checkError" class="mb-4 p-3 rounded-xl bg-red-500/10 border border-red-500/30 text-sm text-red-300">
        {{ t('update.checkFailed') }}
      </div>

      <!-- Actions -->
      <div class="flex flex-wrap gap-2">
        <Button
          :label="t('update.checkAgain')"
          icon="pi pi-refresh"
          severity="secondary"
          @click="checkUpdate"
          :loading="checking"
        />
        <Button
          v-if="updateData.update_available && !updateData.asset_unavailable"
          :label="t('update.install')"
          icon="pi pi-download"
          @click="confirmInstall"
          :disabled="updating"
        />
        <a
          v-if="updateData.release_url"
          :href="updateData.release_url"
          target="_blank"
          rel="noopener"
          class="text-sm text-[var(--accent)] hover:underline self-center ml-2"
        >
          {{ t('update.viewOnGithub') }}
        </a>
      </div>

      <!-- Progress during update -->
      <div v-if="updating || updateStatus.state === 'done'" class="mt-4">
        <ProgressBar :value="updateStatus.progress" />
        <p class="text-sm text-[var(--text-secondary)] mt-2">
          {{ t(`update.state.${updateStatus.state}`) }}
        </p>
        <p v-if="updateStatus.message" class="text-xs text-[var(--text-muted)] mt-1">
          {{ updateStatus.message }}
        </p>
      </div>
    </section>
```

- [ ] **Step 3: Add script logic**

In the `<script setup>` block of SystemInfo.vue, add these imports (merge with existing imports):

```js
import { useToast } from 'primevue/usetoast';
import Badge from 'primevue/badge';
import ProgressBar from 'primevue/progressbar';
import Dialog from 'primevue/dialog';
import axios from '../axios.js';
```

Add reactive state (after existing refs):

```js
const toast = useToast();

const updateData = ref({
  current_version: '',
  latest_version: '',
  update_available: false,
  asset_unavailable: false,
  release_notes: '',
  release_url: '',
  published_at: '',
  os: '',
  arch: '',
  go_version: '',
});
const updateStatus = ref({ state: 'idle', progress: 0, message: '' });
const checking = ref(false);
const updating = ref(false);
const checkError = ref(false);
const showConfirmDialog = ref(false);
let statusPollTimer = null;

const formatDate = (iso) => {
  try {
    return new Date(iso).toLocaleDateString('de-DE', { year: 'numeric', month: 'short', day: 'numeric' });
  } catch {
    return iso;
  }
};

const checkUpdate = async () => {
  checking.value = true;
  checkError.value = false;
  try {
    const res = await axios.get('/api/update/check');
    updateData.value = res.data;
  } catch (err) {
    checkError.value = true;
  } finally {
    checking.value = false;
  }
};

const confirmInstall = () => {
  showConfirmDialog.value = true;
};

const doInstall = async () => {
  showConfirmDialog.value = false;
  updating.value = true;
  try {
    await axios.post('/api/update/perform');
    // Start polling status
    statusPollTimer = setInterval(pollStatus, 1500);
  } catch (err) {
    const msg = err.response?.status === 409
      ? t('update.alreadyRunning')
      : t('update.installFailed', { error: err.response?.data || err.message });
    toast.add({ severity: 'error', summary: t('update.title'), detail: msg, life: 5000 });
    updating.value = false;
  }
};

const pollStatus = async () => {
  try {
    const res = await axios.get('/api/update/status');
    updateStatus.value = res.data;
    if (res.data.state === 'done') {
      clearInterval(statusPollTimer);
      statusPollTimer = null;
      toast.add({ severity: 'success', summary: t('update.title'), detail: t('update.installSuccess'), life: 3000 });
      // Wait for service to restart, then reload
      setTimeout(() => window.location.reload(), 4000);
    } else if (res.data.state === 'error') {
      clearInterval(statusPollTimer);
      statusPollTimer = null;
      updating.value = false;
      toast.add({ severity: 'error', summary: t('update.title'), detail: t('update.installFailed', { error: res.data.error }), life: 8000 });
    }
  } catch (err) {
    // Network error during restart is expected — service is briefly down.
    // Keep polling; it will recover or timeout.
  }
};

// Auto-check on mount
onMounted(() => {
  checkUpdate();
});

onUnmounted(() => {
  if (statusPollTimer) clearInterval(statusPollTimer);
});
```

Add the confirm dialog to the template (before closing `</template>`):
```html
    <Dialog v-model:visible="showConfirmDialog" :header="t('update.confirmTitle')" :modal="true" class="w-11/12 sm:w-full max-w-[440px]">
      <p class="text-sm text-[var(--text-secondary)]">{{ t('update.confirmMessage') }}</p>
      <div class="flex justify-end gap-2 mt-4">
        <Button :label="t('common.cancel')" severity="secondary" @click="showConfirmDialog = false" />
        <Button :label="t('update.install')" icon="pi pi-download" @click="doInstall" />
      </div>
    </Dialog>
```

Ensure `onMounted` and `onUnmounted` are imported from 'vue' (check existing imports — SystemInfo.vue likely already uses them).

- [ ] **Step 4: Build frontend and verify**

Run:
```bash
cd frontend && npm run build
cd .. && rm -rf pkg/web/dist && cp -r frontend/dist pkg/web/dist
```
Expected: Build succeeds, new `SystemInfo-*.js` chunk generated.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/views/SystemInfo.vue pkg/web/dist/
git commit -m "feat(ui): add Update section to System page with check/install/progress"
```

---

## Task 9: Build, deploy, and manual verification

**Files:** None (deployment task)

- [ ] **Step 1: Build the full Linux/amd64 binary**

Run:
```bash
MSYS_NO_PATHCONV=1 MSYS2_ARG_CONV_EXCL="*" docker run --rm -v "/c/Users/basti/Documents/GitHub/modbridge:/src" -w /src -e GOCACHE=/tmp/gocache -e GOMODCACHE=/tmp/gomodcache golang:1.26-bookworm bash -c '
  set -e
  apt-get update -qq > /dev/null 2>&1 && apt-get install -y -qq gcc libc6-dev > /dev/null 2>&1
  CGO_ENABLED=1 GOOS=linux GOARCH=amd64 CC=gcc go build -ldflags="-s -w -X main.Version=2.1.0.0-update -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -trimpath -o /src/modbridge-linux-amd64 .
  echo "BUILD OK"
'
```
Expected: "BUILD OK", binary at `modbridge-linux-amd64`.

- [ ] **Step 2: Verify the new SystemInfo chunk is embedded**

Run: `strings modbridge-linux-amd64 | grep "api/update/check"`
Expected: at least one match confirming the route is compiled in.

- [ ] **Step 3: Deploy to server**

```bash
# Upload
scp modbridge-linux-amd64 root@192.168.178.196:/opt/modbridge/modbridge.new

# Swap and restart via SSH
ssh root@192.168.178.196 'chmod +x /opt/modbridge/modbridge.new && systemctl stop modbridge && sleep 2 && mv /opt/modbridge/modbridge /opt/modbridge/modbridge.old.$(date +%H%M%S) && mv /opt/modbridge/modbridge.new /opt/modbridge/modbridge && systemctl start modbridge && sleep 3 && systemctl is-active modbridge'
```
Expected: `active`

- [ ] **Step 4: Verify in browser**

1. Open `http://192.168.178.196:8080/?v=3#/system`
2. Hard-reload (Ctrl+Shift+R)
3. Log in as Admin
4. Scroll to the new "Update" section
5. Confirm: current version shown, latest version shown, "Aktuell" or "Update verfügbar" badge
6. Click "Erneut prüfen" — should refresh without error
7. Check browser console — no errors

- [ ] **Step 5: Verify API endpoints directly**

Run:
```bash
# Get a session cookie first (reuse browser session or curl login)
curl -s http://192.168.178.196:8080/api/update/check -b "session_token=<from-browser-cookie>"
```
Expected: JSON with `current_version`, `latest_version`, `update_available` fields.

- [ ] **Step 6: Final commit (version bump + cleanup)**

```bash
git add -A
git commit -m "build: embed update module frontend + version bump"
```

---

## Notes for the implementer

- **TDD discipline:** Tasks 1-5 are strictly test-first. Write the test, see it fail, implement, see it pass, commit. Do not skip the "see it fail" step — it catches typos in test setup.
- **No new dependencies:** Everything uses Go stdlib. Do not add `go-selfupdate`, `minio/selfupdate`, or any external library.
- **The `encoding/json` import in `handlers_update.go`:** Remove it if unused — `writeJSON` handles encoding. Keep only what the linter accepts.
- **RBAC:** Verify `rbac.PermSystemRestart` exists in `pkg/rbac/rbac.go:49` before using. If the permission name differs, use the actual constant.
- **main.go constructor changes:** If the `api.NewServer` (or equivalent) constructor doesn't already accept `version` and `buildTime`, add those parameters and update the call site in `main.go` around line 170-180.
- **Concurrency:** `Updater.GetStatus` uses `RLock`. All status writes go through `setStatus`/`setError` which take the write lock. Do not read `u.status` without the lock.
- **Frontend polling:** The 1.5s poll interval in `pollStatus` is deliberate — fast enough to feel responsive, slow enough to avoid hammering. Network errors during the ~5s restart window are expected and silently retried.
