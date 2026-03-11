package admin

import (
	"encoding/json"
	"net/http"

	"nuistagram/internal/httputil"
	"nuistagram/internal/repository"
)

var validRoles = map[string]bool{
	"basic": true,
	"pro":   true,
	"max":   true,
	"admin": true,
}

// Handler handles admin-only HTTP endpoints.
type Handler struct {
	auth  httputil.Authenticator
	users repository.UserRepository
}

func New(auth httputil.Authenticator, users repository.UserRepository) *Handler {
	return &Handler{auth: auth, users: users}
}

func (h *Handler) APIUpdateUserRole(w http.ResponseWriter, r *http.Request) {
	caller := h.auth.CurrentUser(r)
	if caller == nil {
		httputil.JSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if caller.Role != "admin" {
		httputil.JSONError(w, http.StatusForbidden, "forbidden")
		return
	}

	username := r.PathValue("username")
	target, err := h.users.GetByUsername(username)
	if err != nil {
		httputil.JSONError(w, http.StatusNotFound, "user not found")
		return
	}

	var body struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || !validRoles[body.Role] {
		httputil.JSONError(w, http.StatusBadRequest, "invalid role")
		return
	}

	if err := h.users.UpdateRole(target.ID, body.Role); err != nil {
		httputil.JSONError(w, http.StatusInternalServerError, "internal error")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"username": target.Username,
		"role":     body.Role,
	})
}
