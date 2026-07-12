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
		CurrentVersion:   s.updater.CurrentVersion(),
		CurrentBuildTime: s.updater.BuildTime(),
		GoVersion:        s.updater.GoVersion(),
		OS:               s.updater.OS(),
		Arch:             s.updater.Arch(),
	}
	if err != nil {
		// Surface upstream failures (GitHub API down, rate limited, parse
		// error) as 502 so the frontend's checkError path activates. Returning
		// 200 with UpdateAvailable:false here would mask an outage as "no
		// update available".
		http.Error(w, "update check failed: "+err.Error(), http.StatusBadGateway)
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
				s.triggerRestart()
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
