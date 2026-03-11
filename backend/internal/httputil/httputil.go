package httputil

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	jwtpkg "nuistagram/internal/jwt"
	"nuistagram/internal/models"
	"nuistagram/internal/repository"
)

// Authenticator extracts the current user from an HTTP request.
type Authenticator interface {
	CurrentUser(r *http.Request) *models.User
}

func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func JSONError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, map[string]string{"error": msg})
}

func ParseID(s string) int64 {
	id, _ := strconv.ParseInt(s, 10, 64)
	return id
}

func CurrentUserID(user *models.User) int64 {
	if user == nil {
		return 0
	}
	return user.ID
}

// CurrentUser extracts and validates the Bearer token, returning the user or nil.
func CurrentUser(r *http.Request, jwtMgr *jwtpkg.Manager, users repository.UserRepository) *models.User {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil
	}
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	claims, err := jwtMgr.ValidateAccessToken(tokenString)
	if err != nil {
		return nil
	}

	user, err := users.GetByID(claims.UserID)
	if err != nil {
		return nil
	}
	return user
}
