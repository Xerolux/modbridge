package rbac

import (
	"encoding/json"
	"net/http"
)

// Middleware provides HTTP middleware for RBAC
type Middleware struct {
	store *UserStore
}

// NewMiddleware creates a new RBAC middleware
func NewMiddleware(store *UserStore) *Middleware {
	return &Middleware{
		store: store,
	}
}

// RequirePermission creates a middleware that requires a specific permission
func (m *Middleware) RequirePermission(permission Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user from session/token
			user := m.getUserFromRequest(r)
			if user == nil {
				m.sendUnauthorized(w, "Authentication required")
				return
			}

			if !user.Active {
				m.sendForbidden(w, "User account is inactive")
				return
			}

			// Check permission
			hasPermission := false
			for _, p := range user.Permissions {
				if p == permission {
					hasPermission = true
					break
				}
			}

			if !hasPermission {
				m.logAccessDenied(r, user, permission)
				m.sendForbidden(w, "Insufficient permissions")
				return
			}

			// User has permission, proceed with request
			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole creates a middleware that requires a specific role
func (m *Middleware) RequireRole(roles ...Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := m.getUserFromRequest(r)
			if user == nil {
				m.sendUnauthorized(w, "Authentication required")
				return
			}

			// Check if user has one of the required roles
			hasRole := false
			for _, role := range roles {
				if user.Role == role {
					hasRole = true
					break
				}
			}

			if !hasRole {
				m.sendForbidden(w, "Insufficient role privileges")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdmin creates a middleware that requires admin role
func (m *Middleware) RequireAdmin() func(http.Handler) http.Handler {
	return m.RequireRole(RoleAdmin)
}

// APIKeyAuth creates a middleware that authenticates via API key
func (m *Middleware) APIKeyAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get API key from header
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			m.sendUnauthorized(w, "API key required")
			return
		}

		// Validate API key
		user, err := m.store.ValidateAPIToken(apiKey)
		if err != nil {
			if err == ErrInvalidToken {
				m.sendUnauthorized(w, "Invalid API key")
			} else {
				m.sendInternalServerError(w, "Authentication error")
			}
			return
		}

		// Add user to request context
		ctx := ContextWithUser(r.Context(), user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuth tries to get user but doesn't require it
func (m *Middleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := m.getUserFromRequest(r)

		if user != nil {
			ctx := ContextWithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

// getUserFromRequest extracts user from request (session, token, etc.)
func (m *Middleware) getUserFromRequest(r *http.Request) *User {
	// First try to get user from context (already authenticated)
	if user, ok := UserFromContext(r.Context()); ok {
		return user
	}

	// Try API key authentication
	apiKey := r.Header.Get("X-API-Key")
	if apiKey != "" {
		user, err := m.store.ValidateAPIToken(apiKey)
		if err == nil {
			return user
		}
	}

	// Try session cookie
	sessionToken, err := r.Cookie("session_token")
	if err == nil {
		// TODO: Validate session token and get user
		// For now, this would integrate with the auth package
		_ = sessionToken
	}

	// Try basic auth (for API clients)
	username, password, ok := r.BasicAuth()
	if ok {
		authUser, err := m.store.AuthenticateUser(username, password)
		if err == nil {
			return authUser
		}
	}

	return nil
}

// logAccessDenied logs denied access attempts
func (m *Middleware) logAccessDenied(r *http.Request, user *User, permission Permission) {
	// TODO: Implement proper logging
	// For now, this would log to the audit log
}

// sendUnauthorized sends a 401 Unauthorized response
func (m *Middleware) sendUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   "unauthorized",
		"message": message,
	})
}

// sendForbidden sends a 403 Forbidden response
func (m *Middleware) sendForbidden(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   "forbidden",
		"message": message,
	})
}

// sendInternalServerError sends a 500 Internal Server Error response
func (m *Middleware) sendInternalServerError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   "internal_server_error",
		"message": message,
	})
}

// PermissionChecker provides permission checking for HTTP handlers
type PermissionChecker struct {
	store *UserStore
}

// NewPermissionChecker creates a new permission checker
func NewPermissionChecker(store *UserStore) *PermissionChecker {
	return &PermissionChecker{
		store: store,
	}
}

// Check checks if the current user has the required permission
func (pc *PermissionChecker) Check(r *http.Request, permission Permission) bool {
	user, ok := UserFromContext(r.Context())
	if !ok {
		return false
	}

	if !user.Active {
		return false
	}

	for _, p := range user.Permissions {
		if p == permission {
			return true
		}
	}

	return false
}

// CheckAny checks if the user has any of the required permissions
func (pc *PermissionChecker) CheckAny(r *http.Request, permissions ...Permission) bool {
	user, ok := UserFromContext(r.Context())
	if !ok {
		return false
	}

	if !user.Active {
		return false
	}

	for _, requiredPerm := range permissions {
		for _, userPerm := range user.Permissions {
			if userPerm == requiredPerm {
				return true
			}
		}
	}

	return false
}

// CheckAll checks if the user has all of the required permissions
func (pc *PermissionChecker) CheckAll(r *http.Request, permissions ...Permission) bool {
	user, ok := UserFromContext(r.Context())
	if !ok {
		return false
	}

	if !user.Active {
		return false
	}

	for _, requiredPerm := range permissions {
		hasPerm := false
		for _, userPerm := range user.Permissions {
			if userPerm == requiredPerm {
				hasPerm = true
				break
			}
		}
		if !hasPerm {
			return false
		}
	}

	return true
}

// GetCurrentUser returns the current user from the request
func (pc *PermissionChecker) GetCurrentUser(r *http.Request) (*User, bool) {
	return UserFromContext(r.Context())
}

// RequireUserID checks if the current user matches the specified user ID
// or if the user is an admin
func (pc *PermissionChecker) RequireUserID(r *http.Request, userID string) bool {
	user, ok := UserFromContext(r.Context())
	if !ok {
		return false
	}

	// Admin can access any user
	if user.Role == RoleAdmin {
		return true
	}

	// User can access their own resources
	return user.ID == userID
}

// HasRole checks if the current user has a specific role
func (pc *PermissionChecker) HasRole(r *http.Request, role Role) bool {
	user, ok := UserFromContext(r.Context())
	if !ok {
		return false
	}

	return user.Role == role
}

// IsAdmin checks if the current user is an admin
func (pc *PermissionChecker) IsAdmin(r *http.Request) bool {
	return pc.HasRole(r, RoleAdmin)
}
