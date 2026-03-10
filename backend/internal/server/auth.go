package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"regexp"

	"golang.org/x/crypto/bcrypt"
)

const (
	passwordMinLength = 8
)

func validatePassword(password string) error {
	if len(password) < passwordMinLength {
		return &validationError{"Password must be at least 8 characters long"}
	}
	if matched, _ := regexp.MatchString(`[A-Z]`, password); !matched {
		return &validationError{"Password must contain at least one uppercase letter"}
	}
	if matched, _ := regexp.MatchString(`[a-z]`, password); !matched {
		return &validationError{"Password must contain at least one lowercase letter"}
	}
	if matched, _ := regexp.MatchString(`[0-9]`, password); !matched {
		return &validationError{"Password must contain at least one digit"}
	}
	return nil
}

type validationError struct {
	Message string
}

func (e *validationError) Error() string {
	return e.Message
}

func (s *Server) Register(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		jsonError(w, http.StatusBadRequest, "Username and password required")
		return
	}

	if len(username) < 3 || len(username) > 50 {
		jsonError(w, http.StatusBadRequest, "Username must be between 3 and 50 characters")
		return
	}

	if err := validatePassword(password); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "Internal error")
		return
	}

	userID, err := s.Repos.Users.Create(username, string(hash))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "Username already exists")
		return
	}

	accessToken, err := s.JWT.GenerateAccessToken(userID, username)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "Internal error")
		return
	}
	refreshToken, err := s.JWT.GenerateRefreshToken(userID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "Internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    int(s.JWT.AccessTokenTTL().Seconds()),
		"token_type":    "Bearer",
	})
}

func (s *Server) Login(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	user, err := s.Repos.Users.GetByUsername(username)
	if err == sql.ErrNoRows {
		jsonError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "Internal error")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		jsonError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	accessToken, err := s.JWT.GenerateAccessToken(user.ID, user.Username)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "Internal error")
		return
	}
	refreshToken, err := s.JWT.GenerateRefreshToken(user.ID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "Internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    int(s.JWT.AccessTokenTTL().Seconds()),
		"token_type":    "Bearer",
	})
}

func (s *Server) Logout(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.RefreshToken != "" {
		s.JWT.RevokeRefreshToken(body.RefreshToken)
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) Refresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RefreshToken == "" {
		jsonError(w, http.StatusBadRequest, "refresh_token required")
		return
	}

	userID, err := s.JWT.ConsumeRefreshToken(body.RefreshToken)
	if err != nil {
		jsonError(w, http.StatusUnauthorized, "Invalid or expired refresh token")
		return
	}

	user, err := s.Repos.Users.GetByID(userID)
	if err != nil {
		jsonError(w, http.StatusUnauthorized, "User not found")
		return
	}

	accessToken, err := s.JWT.GenerateAccessToken(user.ID, user.Username)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "Internal error")
		return
	}
	newRefresh, err := s.JWT.GenerateRefreshToken(user.ID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "Internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  accessToken,
		"refresh_token": newRefresh,
		"expires_in":    int(s.JWT.AccessTokenTTL().Seconds()),
		"token_type":    "Bearer",
	})
}
