package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"nuistagram/internal/models"
)

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func parseID(s string) int64 {
	id, _ := strconv.ParseInt(s, 10, 64)
	return id
}

func (s *Server) currentUser(r *http.Request) *models.User {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil
	}
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	claims, err := s.JWT.ValidateAccessToken(tokenString)
	if err != nil {
		return nil
	}

	user, err := s.Repos.Users.GetByID(claims.UserID)
	if err != nil {
		return nil
	}
	return user
}

func currentUserID(user *models.User) int64 {
	if user == nil {
		return 0
	}
	return user.ID
}
