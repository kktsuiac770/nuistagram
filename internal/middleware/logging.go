package middleware

import (
	"log/slog"
	"net/http"
	"nuistagram/internal/logging"
	"time"
)

// Chain applies middlewares in reverse order so the first one listed runs first.
func Chain(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

// ResponseWriter wraps http.ResponseWriter to capture the status code.
type ResponseWriter struct {
	http.ResponseWriter
	Status      int
	WroteHeader bool
}

func (w *ResponseWriter) WriteHeader(code int) {
	if w.WroteHeader {
		return
	}
	w.Status = code
	w.WroteHeader = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *ResponseWriter) Write(b []byte) (int, error) {
	if !w.WroteHeader {
		w.Status = http.StatusOK
		w.WroteHeader = true
	}
	return w.ResponseWriter.Write(b)
}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &ResponseWriter{ResponseWriter: w, Status: http.StatusOK}

		requestID := GetRequestID(r.Context())
		log := logging.With(
			slog.String("request_id", requestID),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("remote_addr", r.RemoteAddr),
		)

		log.Debug("request started")

		ctx := logging.ContextWithLogger(r.Context(), log)
		next.ServeHTTP(rw, r.WithContext(ctx))

		duration := time.Since(start)
		status := rw.Status

		logLevel := slog.LevelInfo
		if status >= 400 && status < 500 {
			logLevel = slog.LevelWarn
		} else if status >= 500 {
			logLevel = slog.LevelError
		}

		log.LogAttrs(ctx, logLevel, "request completed",
			slog.Int("status", status),
			slog.Duration("latency", duration),
		)
	})
}
