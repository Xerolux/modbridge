// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package api

import (
	"encoding/csv"
	"fmt"
	"modbridge/pkg/database"
	"modbridge/pkg/rbac"
	"net/http"
)

// handleDevices returns all tracked devices.
func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request) {
	permissionByMethod := map[string]rbac.Permission{
		http.MethodGet: rbac.PermDeviceView,
		http.MethodPut: rbac.PermDeviceEdit,
	}

	permission, exists := permissionByMethod[r.Method]
	if !exists {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.requirePermission(w, r, permission) == nil {
		return
	}

	if r.Method == http.MethodGet {
		devices := s.mgr.GetDevices()
		w.Header().Set("Content-Type", "application/json")
		s.writeJSON(w, devices)
		return
	}

	if r.Method == http.MethodPut {
		var req struct {
			IP   string `json:"ip"`
			Name string `json:"name"`
		}
		if err := decodeJSON(w, r, &req); err != nil {
			writeJSONDecodeError(w, err)
			return
		}

		if err := s.mgr.SetDeviceName(req.IP, req.Name); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		return
	}
}

// handleDeviceHistory returns connection history for devices.
func (s *Server) handleDeviceHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.requirePermission(w, r, rbac.PermDeviceView) == nil {
		return
	}

	// Get query parameters
	deviceIP := r.URL.Query().Get("device_ip")
	proxyID := r.URL.Query().Get("proxy_id")
	limit := 100                          // default limit
	format := r.URL.Query().Get("format") // csv or json

	var history interface{}
	var err error

	if deviceIP != "" {
		// Get history for specific device
		history, err = s.mgr.GetConnectionHistory(deviceIP, limit)
	} else {
		// Get all history, optionally filtered by proxy
		history, err = s.mgr.GetAllConnectionHistory(proxyID, limit)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if format == "csv" {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=device_history.csv")

		csvWriter := csv.NewWriter(w)
		if err := csvWriter.Write([]string{"IP", "Proxy ID", "Connected At", "Request Count"}); err != nil {
			http.Error(w, "Failed to write CSV header", http.StatusInternalServerError)
			return
		}
		if entries, ok := history.([]*database.ConnectionHistoryEntry); ok {
			for _, entry := range entries {
				if err := csvWriter.Write([]string{
					entry.DeviceIP,
					entry.ProxyID,
					entry.ConnectedAt.Format("2006-01-02 15:04:05"),
					fmt.Sprintf("%d", entry.RequestCount),
				}); err != nil {
					s.log.Error("API", fmt.Sprintf("Failed to write CSV row: %v", err))
					break
				}
			}
		}
		csvWriter.Flush()
		return
	}

	w.Header().Set("Content-Type", "application/json")
	s.writeJSON(w, history)
}
