package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"nuistagram/internal/database"
)

func (s *Server) Healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) Readyz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if err := database.HealthCheck(s.db()); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "not ready",
			"checks": map[string]string{
				"database": err.Error(),
			},
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ready",
		"checks": map[string]string{
			"database": "ok",
		},
	})
}

func (s *Server) db() *sql.DB {
	// Access DB through the photo repository's underlying connection.
	// We store a reference on the server for health checks.
	return s.DB
}

func (s *Server) registerHealthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", s.Healthz)
	mux.HandleFunc("/readyz", s.Readyz)
	// TODO: add metrics endpoint
}
