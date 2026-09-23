package logger

import (
	"context"
	"log/slog"
)

type contextKey struct{}

var loggerKey contextKey

func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}

func FromContext(ctx context.Context) (*slog.Logger, bool) {
	l, ok := ctx.Value(loggerKey).(*slog.Logger)
	return l, ok
}
