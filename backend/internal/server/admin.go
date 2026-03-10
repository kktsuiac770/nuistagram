package server

import (
	"encoding/json"
	"net/http"
)

var validRoles = map[string]bool{
	"basic": true,
	"pro":   true,
	"max":   true,
	"admin": true,
}

func (s *Server) APIUpdateUserRole(w http.ResponseWriter, r *http.Request) {
	caller := s.currentUser(r)
	if caller == nil {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if caller.Role != "admin" {
		jsonError(w, http.StatusForbidden, "forbidden")
		return
	}

	username := r.PathValue("username")
	target, err := s.Repos.Users.GetByUsername(username)
	if err != nil {
		jsonError(w, http.StatusNotFound, "user not found")
		return
	}

	var body struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || !validRoles[body.Role] {
		jsonError(w, http.StatusBadRequest, "invalid role")
		return
	}

	if err := s.Repos.Users.UpdateRole(target.ID, body.Role); err != nil {
		jsonError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"username": target.Username,
		"role":     body.Role,
	})
}
