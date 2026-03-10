package middleware

import (
	"fmt"
	"net/http"
	"regexp"
	"time"

	"nuistagram/internal/monitoring/metrics"
)

var rePathNumericID = regexp.MustCompile(`/\d+`)

// normalizePath replaces numeric path segments with /:id to prevent
// cardinality explosion in metric labels (e.g. /api/photo/123 → /api/photo/:id).
func normalizePath(path string) string {
	return rePathNumericID.ReplaceAllString(path, "/:id")
}

// Metrics records Prometheus HTTP metrics for every request.
// The /metrics, /healthz, and /readyz paths are skipped to avoid noise.
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/metrics" || path == "/healthz" || path == "/readyz" {
			next.ServeHTTP(w, r)
			return
		}

		metrics.HTTPRequestsInFlight.Inc()
		defer metrics.HTTPRequestsInFlight.Dec()

		normalizedPath := normalizePath(path)
		start := time.Now()
		rw := &ResponseWriter{ResponseWriter: w, Status: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start).Seconds()
		status := fmt.Sprintf("%d", rw.Status)

		metrics.HTTPRequestsTotal.WithLabelValues(r.Method, normalizedPath, status).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(r.Method, normalizedPath).Observe(duration)
	})
}
