package server

import (
	"encoding/json"
	"html/template"
	"net/http"
	"nuistagram/internal/models"
	"strconv"
)

var templates *template.Template

var funcMap = template.FuncMap{
	"sub": func(a, b int) int { return a - b },
	"add": func(a, b int) int { return a + b },
}

func (s *Server) initTemplates() error {
	var err error
	templates, err = template.New("").Funcs(funcMap).ParseGlob("templates/*.html")
	return err
}

func renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	if err := templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

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
	cookie, err := r.Cookie("session")
	if err != nil {
		return nil
	}

	if !s.Sessions.IsValid(cookie.Value) {
		return nil
	}

	userID := s.Sessions.GetUserID(cookie.Value)
	if userID == 0 {
		return nil
	}

	user, err := s.Repos.Users.GetByID(userID)
	if err != nil {
		return nil
	}
	return user
}

func (s *Server) sessionToken(r *http.Request) string {
	cookie, err := r.Cookie("session")
	if err != nil {
		return ""
	}
	return cookie.Value
}

func currentUserID(user *models.User) int64 {
	if user == nil {
		return 0
	}
	return user.ID
}
