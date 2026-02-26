package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"nuistagram/internal/logging"
	"runtime/debug"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				requestID := GetRequestID(r.Context())
				stack := string(debug.Stack())

				logging.With(
					slog.String("request_id", requestID),
					slog.String("path", r.URL.Path),
					slog.String("method", r.Method),
				).Error("panic recovered",
					slog.Any("error", err),
					slog.String("stack", stack),
				)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, `{"error":"Internal server error","request_id":"%s"}`, requestID)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
