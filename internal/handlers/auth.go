package handlers

import (
	"database/sql"
	"net/http"
	"nuistagram/internal/models"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func Register(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		http.Error(w, `{"error": "Username and password required"}`, http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	_, err = Repos.Users.Create(username, string(hash))
	if err != nil {
		http.Error(w, `{"error": "Username already exists"}`, http.StatusBadRequest)
		return
	}

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
		Expires:  time.Now().Add(24 * time.Hour),
	})

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"success": true}`))
}

func Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
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
