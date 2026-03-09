package server

import (
	"database/sql"
	"net/http"
	"regexp"
	"time"

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

	session := s.Sessions.Create(userID)
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    session,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.secureCookie,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(24 * time.Hour),
	})

	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
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

	session := s.Sessions.Create(user.ID)
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    session,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.secureCookie,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(24 * time.Hour),
	})

	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		s.Sessions.Delete(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   s.secureCookie,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}
