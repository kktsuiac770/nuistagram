package middleware

import (
	"log/slog"
	"net/http"
	"nuistagram/internal/logging"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *responseWriter) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}
	w.status = code
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.status = http.StatusOK
		w.wroteHeader = true
	}
	return w.ResponseWriter.Write(b)
}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

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
		status := rw.status

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
