package handlers

import (
	"net/http"
)

func GetCSRFToken(w http.ResponseWriter, r *http.Request) {
	sessionToken := GetSessionToken(r)
	if sessionToken == "" {
		http.Error(w, `{"error": "Not authenticated"}`, http.StatusUnauthorized)
		return
	}

	csrfToken := GenerateCSRFToken(sessionToken)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"csrf_token": "` + csrfToken + `"}`))
}

func CSRFMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
			next(w, r)
			return
		}

		sessionToken := GetSessionToken(r)
		if sessionToken == "" {
			http.Error(w, `{"error": "Not authenticated"}`, http.StatusUnauthorized)
			return
		}

		csrfToken := r.Header.Get("X-CSRF-Token")
		if csrfToken == "" {
			csrfToken = r.FormValue("csrf_token")
		}

		if csrfToken == "" || !ValidateCSRFToken(sessionToken, csrfToken) {
			http.Error(w, `{"error": "Invalid CSRF token"}`, http.StatusForbidden)
			return
		}

		next(w, r)
	}
}

func CSRFMiddlewareFunc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}

		sessionToken := GetSessionToken(r)
		if sessionToken == "" {
			next.ServeHTTP(w, r)
			return
		}

		csrfToken := r.Header.Get("X-CSRF-Token")
		if csrfToken == "" {
			csrfToken = r.FormValue("csrf_token")
		}

		if csrfToken == "" || !ValidateCSRFToken(sessionToken, csrfToken) {
			http.Error(w, `{"error": "Invalid CSRF token"}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
