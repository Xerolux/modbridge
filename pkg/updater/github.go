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
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"` // release notes / changelog
	HTMLURL     string    `json:"html_url"`
	PublishedAt time.Time `json:"published_at"`
	Prerelease  bool      `json:"prerelease"`
	Assets      []Asset   `json:"assets"`
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
