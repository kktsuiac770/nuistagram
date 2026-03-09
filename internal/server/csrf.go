package server

import (
	"net/http"
)

func (s *Server) GetCSRFToken(w http.ResponseWriter, r *http.Request) {
	sessionToken := s.sessionToken(r)
	if sessionToken == "" {
		jsonError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	csrfToken := s.Sessions.GenerateCSRFToken(sessionToken)
	writeJSON(w, http.StatusOK, map[string]string{"csrf_token": csrfToken})
}

func (s *Server) csrfMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
			next(w, r)
			return
		}

		sessionToken := s.sessionToken(r)
		if sessionToken == "" {
			jsonError(w, http.StatusUnauthorized, "Not authenticated")
			return
		}

		csrfToken := r.Header.Get("X-CSRF-Token")
		if csrfToken == "" {
			csrfToken = r.FormValue("csrf_token")
		}

		if csrfToken == "" || !s.Sessions.ValidateCSRFToken(sessionToken, csrfToken) {
			jsonError(w, http.StatusForbidden, "Invalid CSRF token")
			return
		}

		next(w, r)
	}
}
