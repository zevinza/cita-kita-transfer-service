package logging

import (
	"context"
	"log/slog"
	"os"
)

//go:generate go run go.uber.org/mock/mockgen@latest -source=logger.go -destination=logger_mock.go -package=logging Logger
type Logger interface {
	Error(ctx context.Context, msg string, args ...any)
	Warn(ctx context.Context, msg string, args ...any)
	Info(ctx context.Context, msg string, args ...any)
	Debug(ctx context.Context, msg string, args ...any)
}

type logger struct {
	Logger *slog.Logger
}

func NewLogger() Logger {
	return &logger{
		Logger: slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}
}

func (l *logger) Error(ctx context.Context, msg string, args ...any) {
	l.Logger.ErrorContext(ctx, msg, args...)
}

func (l *logger) Warn(ctx context.Context, msg string, args ...any) {
	l.Logger.WarnContext(ctx, msg, args...)
}

func (l *logger) Info(ctx context.Context, msg string, args ...any) {
	l.Logger.InfoContext(ctx, msg, args...)
}

func (l *logger) Debug(ctx context.Context, msg string, args ...any) {
	l.Logger.DebugContext(ctx, msg, args...)
}
