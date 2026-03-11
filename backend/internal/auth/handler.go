package auth

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"nuistagram/internal/httputil"
	jwtpkg "nuistagram/internal/jwt"
	"nuistagram/internal/models"
	"nuistagram/internal/monitoring/metrics"
	"nuistagram/internal/repository"
)

const passwordMinLength = 8

type validationError struct {
	Message string
}

func (e *validationError) Error() string {
	return e.Message
}

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

// Handler handles authentication endpoints and implements httputil.Authenticator.
type Handler struct {
	users repository.UserRepository
	jwt   *jwtpkg.Manager
}

func New(users repository.UserRepository, jwt *jwtpkg.Manager) *Handler {
	return &Handler{users: users, jwt: jwt}
}

// CurrentUser implements httputil.Authenticator.
func (h *Handler) CurrentUser(r *http.Request) *models.User {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil
	}
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	claims, err := h.jwt.ValidateAccessToken(tokenString)
	if err != nil {
		return nil
	}

	user, err := h.users.GetByID(claims.UserID)
	if err != nil {
		return nil
	}
	return user
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		httputil.JSONError(w, http.StatusBadRequest, "Username and password required")
		return
	}

	if len(username) < 3 || len(username) > 50 {
		httputil.JSONError(w, http.StatusBadRequest, "Username must be between 3 and 50 characters")
		return
	}

	if err := validatePassword(password); err != nil {
		httputil.JSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		httputil.JSONError(w, http.StatusInternalServerError, "Internal error")
		return
	}

	userID, err := h.users.Create(username, string(hash))
	if err != nil {
		metrics.RegistrationsTotal.WithLabelValues("failure").Inc()
		httputil.JSONError(w, http.StatusBadRequest, "Username already exists")
		return
	}
	metrics.RegistrationsTotal.WithLabelValues("success").Inc()

	accessToken, err := h.jwt.GenerateAccessToken(userID, username)
	if err != nil {
		httputil.JSONError(w, http.StatusInternalServerError, "Internal error")
		return
	}
	refreshToken, err := h.jwt.GenerateRefreshToken(userID)
	if err != nil {
		httputil.JSONError(w, http.StatusInternalServerError, "Internal error")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    int(h.jwt.AccessTokenTTL().Seconds()),
		"token_type":    "Bearer",
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	user, err := h.users.GetByUsername(username)
	if err == sql.ErrNoRows {
		metrics.LoginsTotal.WithLabelValues("failure").Inc()
		httputil.JSONError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}
	if err != nil {
		httputil.JSONError(w, http.StatusInternalServerError, "Internal error")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		metrics.LoginsTotal.WithLabelValues("failure").Inc()
		httputil.JSONError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}
	metrics.LoginsTotal.WithLabelValues("success").Inc()

	accessToken, err := h.jwt.GenerateAccessToken(user.ID, user.Username)
	if err != nil {
		httputil.JSONError(w, http.StatusInternalServerError, "Internal error")
		return
	}
	refreshToken, err := h.jwt.GenerateRefreshToken(user.ID)
	if err != nil {
		httputil.JSONError(w, http.StatusInternalServerError, "Internal error")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    int(h.jwt.AccessTokenTTL().Seconds()),
		"token_type":    "Bearer",
	})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.RefreshToken != "" {
		h.jwt.RevokeRefreshToken(body.RefreshToken)
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RefreshToken == "" {
		httputil.JSONError(w, http.StatusBadRequest, "refresh_token required")
		return
	}

	userID, err := h.jwt.ConsumeRefreshToken(body.RefreshToken)
	if err != nil {
		httputil.JSONError(w, http.StatusUnauthorized, "Invalid or expired refresh token")
		return
	}

	user, err := h.users.GetByID(userID)
	if err != nil {
		httputil.JSONError(w, http.StatusUnauthorized, "User not found")
		return
	}

	accessToken, err := h.jwt.GenerateAccessToken(user.ID, user.Username)
	if err != nil {
		httputil.JSONError(w, http.StatusInternalServerError, "Internal error")
		return
	}
	newRefresh, err := h.jwt.GenerateRefreshToken(user.ID)
	if err != nil {
		httputil.JSONError(w, http.StatusInternalServerError, "Internal error")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"access_token":  accessToken,
		"refresh_token": newRefresh,
		"expires_in":    int(h.jwt.AccessTokenTTL().Seconds()),
		"token_type":    "Bearer",
	})
}
