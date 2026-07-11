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

	// Phase 3: Atomic swap — locate the running binary so the temp file is
	// renamed atomically over it (same-directory rename required by SwapBinary).
	u.setStatus(StateSwapping, 90, "swapping binary")
	execPath, execErr := os.Executable()
	if execErr != nil {
		u.setError("could not locate current executable: " + execErr.Error())
		os.Remove(tempPath)
		return
	}
	execPath, _ = filepath.EvalSymlinks(execPath)
	if _, err := SwapBinary(tempPath, execPath); err != nil {
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
