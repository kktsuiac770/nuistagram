package middleware

import (
	"net/http"
	"nuistagram/internal/tracing"
)

func Tracing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracing.Start(r.Context(), r.Method+" "+r.URL.Path,
			"http.method", r.Method,
			"http.url", r.URL.String(),
			"http.host", r.Host,
		)
		defer span.End()

		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r.WithContext(ctx))

		span.SetAttribute("http.status_code", rw.status)
		if rw.status >= 400 {
			span.SetStatus("ERROR", http.StatusText(rw.status))
		} else {
			span.SetStatus("OK", "")
		}
	})
}
