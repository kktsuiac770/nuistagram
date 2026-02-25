package handlers

import (
	"database/sql"
	"net/http"
	"nuistagram/internal/models"
	"regexp"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	passwordMinLength     = 8
	passwordRequiresUpper = true
	passwordRequiresLower = true
	passwordRequiresDigit = true
	secureCookie          = false
)

func SetSecureCookie(secure bool) {
	secureCookie = secure
}

type PasswordValidationError struct {
	Message string
}

func (e PasswordValidationError) Error() string {
	return e.Message
}

func validatePassword(password string) error {
	if len(password) < passwordMinLength {
		return PasswordValidationError{Message: "Password must be at least 8 characters long"}
	}

	if passwordRequiresUpper {
		matched, _ := regexp.MatchString(`[A-Z]`, password)
		if !matched {
			return PasswordValidationError{Message: "Password must contain at least one uppercase letter"}
		}
	}

	if passwordRequiresLower {
		matched, _ := regexp.MatchString(`[a-z]`, password)
		if !matched {
			return PasswordValidationError{Message: "Password must contain at least one lowercase letter"}
		}
	}

	if passwordRequiresDigit {
		matched, _ := regexp.MatchString(`[0-9]`, password)
		if !matched {
			return PasswordValidationError{Message: "Password must contain at least one digit"}
		}
	}

	return nil
}

func Register(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		http.Error(w, `{"error": "Username and password required"}`, http.StatusBadRequest)
		return
	}

	if len(username) < 3 || len(username) > 50 {
		http.Error(w, `{"error": "Username must be between 3 and 50 characters"}`, http.StatusBadRequest)
		return
	}

	if err := validatePassword(password); err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	userID, err := Repos.Users.Create(username, string(hash))
	if err != nil {
		http.Error(w, `{"error": "Username already exists"}`, http.StatusBadRequest)
		return
	}

	session := CreateSession(userID)
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    session,
		Path:     "/",
		HttpOnly: true,
		Secure:   secureCookie,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(24 * time.Hour),
	})

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"success": true}`))
}

func Login(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	user, err := Repos.Users.GetByUsername(username)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error": "Invalid credentials"}`, http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		http.Error(w, `{"error": "Invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	session := CreateSession(user.ID)
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    session,
		Path:     "/",
		HttpOnly: true,
		Secure:   secureCookie,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(24 * time.Hour),
	})

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"success": true}`))
}

func Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		DeleteSession(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secureCookie,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"success": true}`))
}

func GetCurrentUser(r *http.Request) *models.User {
	cookie, err := r.Cookie("session")
	if err != nil {
		return nil
	}

	if !IsSessionValid(cookie.Value) {
		return nil
	}

	userID := GetUserIDFromSession(cookie.Value)
	if userID == 0 {
		return nil
	}

	user, err := Repos.Users.GetByID(userID)
	if err != nil {
		return nil
	}

	return user
}

func GetSessionToken(r *http.Request) string {
	cookie, err := r.Cookie("session")
	if err != nil {
		return ""
	}
	return cookie.Value
}
