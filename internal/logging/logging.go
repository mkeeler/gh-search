package logging

import (
	"context"
	"io"
	"log/slog"
)

const (
	LevelTrace slog.Level = -8
)

type loggingCtxKey struct{}

var noopLogger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))

func WithContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggingCtxKey{}, logger)
}

func FromContext(ctx context.Context) *slog.Logger {
	val := ctx.Value(loggingCtxKey{})
	if val == nil {
		return slog.Default()
	}

	if logger, ok := val.(*slog.Logger); ok {
		return logger
	}

	return slog.Default()
}
