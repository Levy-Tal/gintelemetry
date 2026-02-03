package gintelemetry

import (
	"context"
	"log/slog"
)

var logger *slog.Logger

// Log levels.
const (
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)

// Level represents a log level.
type Level = slog.Level

func setLogger(otelLogger *slog.Logger, level Level) {
	handler := &levelFilterHandler{
		handler: otelLogger.Handler(),
		level:   level,
	}
	logger = slog.New(handler)
	slog.SetDefault(logger)
}

// levelFilterHandler wraps an slog.Handler and filters by level.
type levelFilterHandler struct {
	handler slog.Handler
	level   Level
}

func (h *levelFilterHandler) Enabled(ctx context.Context, level Level) bool {
	return level >= h.level
}

func (h *levelFilterHandler) Handle(ctx context.Context, r slog.Record) error {
	return h.handler.Handle(ctx, r)
}

func (h *levelFilterHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &levelFilterHandler{
		handler: h.handler.WithAttrs(attrs),
		level:   h.level,
	}
}

func (h *levelFilterHandler) WithGroup(name string) slog.Handler {
	return &levelFilterHandler{
		handler: h.handler.WithGroup(name),
		level:   h.level,
	}
}

// LogAPI provides structured logging functions with automatic trace correlation.
type LogAPI struct{}

// Log is the namespace for all logging operations.
var Log = LogAPI{}

// Logger returns the configured slog logger with trace correlation.
func (LogAPI) Logger() *slog.Logger {
	return logger
}

// Info logs an informational message.
func (LogAPI) Info(ctx context.Context, msg string, args ...any) {
	logger.InfoContext(ctx, msg, args...)
}

// Warn logs a warning message.
func (LogAPI) Warn(ctx context.Context, msg string, args ...any) {
	logger.WarnContext(ctx, msg, args...)
}

// Error logs an error message.
func (LogAPI) Error(ctx context.Context, msg string, args ...any) {
	logger.ErrorContext(ctx, msg, args...)
}

// Debug logs a debug message.
func (LogAPI) Debug(ctx context.Context, msg string, args ...any) {
	logger.DebugContext(ctx, msg, args...)
}

// With returns a new logger with the given attributes pre-set.
func (LogAPI) With(args ...any) *slog.Logger {
	return logger.With(args...)
}

// WithGroup returns a new logger with the given group name.
func (LogAPI) WithGroup(name string) *slog.Logger {
	return logger.WithGroup(name)
}
