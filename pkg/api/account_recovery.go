package api

import (
	"net/http"
	"strings"
)

func (s *Server) handleAccountRecovery(w http.ResponseWriter, r *http.Request) {
	if s.userMgr == nil {
		http.Error(w, "Account recovery is unavailable", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		available, err := s.userMgr.AccountRecoveryAvailable()
		if err != nil {
			http.Error(w, "Could not check account recovery", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		s.writeJSON(w, map[string]bool{"available": available})

	case http.MethodPost:
		var req struct {
			Token       string `json:"token"`
			NewUsername string `json:"new_username"`
			NewPassword string `json:"new_password"`
		}
		if err := decodeJSON(w, r, &req); err != nil {
			writeJSONDecodeError(w, err)
			return
		}
		userID, err := s.userMgr.RecoverAdminAccount(req.Token, req.NewUsername, req.NewPassword)
		if err != nil {
			// Recovery errors intentionally do not disclose the target identity.
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if s.auth != nil {
			s.auth.InvalidateUserSessions(userID)
		}
		if s.auditor != nil {
			ip, ua := requestMeta(r)
			s.auditor.LogUserAction("user.recovered", userID, userID, strings.TrimSpace(req.NewUsername), "local recovery code used", ip, ua, true)
		}
		w.Header().Set("Content-Type", "application/json")
		s.writeJSON(w, map[string]string{"status": "ok"})

	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
