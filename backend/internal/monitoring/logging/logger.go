package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
)

type ctxKey struct{}

var logger *slog.Logger

func Init(env string, w io.Writer) {
	if w == nil {
		w = os.Stdout
	}
	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: slog.LevelDebug}

	if env == "production" {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}

	logger = slog.New(handler)
	slog.SetDefault(logger)
}

func L() *slog.Logger {
	if logger == nil {
		return slog.Default()
	}
	return logger
}

func With(args ...any) *slog.Logger {
	return L().With(args...)
}

func WithContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		return l
	}
	return L()
}

func ContextWithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

func Debug(msg string, args ...any) { L().Debug(msg, args...) }
func Info(msg string, args ...any)  { L().Info(msg, args...) }
func Warn(msg string, args ...any)  { L().Warn(msg, args...) }
func Error(msg string, args ...any) { L().Error(msg, args...) }

func DebugCtx(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).DebugContext(ctx, msg, args...)
}
func InfoCtx(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).InfoContext(ctx, msg, args...)
}
func WarnCtx(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).WarnContext(ctx, msg, args...)
}
func ErrorCtx(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).ErrorContext(ctx, msg, args...)
}
