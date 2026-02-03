package gintelemetry

import (
	"context"
	"log/slog"
)

const (
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)

type Level = slog.Level

func applyLevelFilter(otelLogger *slog.Logger, level Level) *slog.Logger {
	handler := &levelFilterHandler{
		handler: otelLogger.Handler(),
		level:   level,
	}
	return slog.New(handler)
}

type levelFilterHandler struct {
	handler slog.Handler
	level   Level
}

func (h *levelFilterHandler) Enabled(_ context.Context, level Level) bool {
	return level >= h.level
}

func (h *levelFilterHandler) Handle(ctx context.Context, r slog.Record) error {
	return h.handler.Handle(ctx, r)
}

func (h *levelFilterHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &levelFilterHandler{handler: h.handler.WithAttrs(attrs), level: h.level}
}

func (h *levelFilterHandler) WithGroup(name string) slog.Handler {
	return &levelFilterHandler{handler: h.handler.WithGroup(name), level: h.level}
}

type LogAPI struct {
	logger *slog.Logger
}

func (l LogAPI) Logger() *slog.Logger {
	return l.logger
}

// Info logs an informational message with trace correlation.
// The context should contain an active span from the request or manually created span.
// If ctx has no span, logs will still be recorded but without trace correlation.
// Using context.Background() is safe but loses correlation benefits.
//
// Example:
//
//	tel.Log().Info(ctx, "user logged in", "user_id", userID, "ip", clientIP)
func (l LogAPI) Info(ctx context.Context, msg string, args ...any) {
	if l.logger != nil {
		l.logger.InfoContext(ctx, msg, args...)
	}
}

// Warn logs a warning message with trace correlation.
// The context should contain an active span from the request or manually created span.
// If ctx has no span, logs will still be recorded but without trace correlation.
// Using context.Background() is safe but loses correlation benefits.
//
// Example:
//
//	tel.Log().Warn(ctx, "rate limit approaching", "current", count, "limit", maxCount)
func (l LogAPI) Warn(ctx context.Context, msg string, args ...any) {
	if l.logger != nil {
		l.logger.WarnContext(ctx, msg, args...)
	}
}

// Error logs an error message with trace correlation.
// The context should contain an active span from the request or manually created span.
// If ctx has no span, logs will still be recorded but without trace correlation.
// Using context.Background() is safe but loses correlation benefits.
//
// Example:
//
//	tel.Log().Error(ctx, "database query failed", "error", err.Error(), "query", query)
func (l LogAPI) Error(ctx context.Context, msg string, args ...any) {
	if l.logger != nil {
		l.logger.ErrorContext(ctx, msg, args...)
	}
}

// Debug logs a debug message with trace correlation.
// The context should contain an active span from the request or manually created span.
// If ctx has no span, logs will still be recorded but without trace correlation.
// Using context.Background() is safe but loses correlation benefits.
//
// Example:
//
//	tel.Log().Debug(ctx, "cache hit", "key", cacheKey, "ttl", ttl)
func (l LogAPI) Debug(ctx context.Context, msg string, args ...any) {
	if l.logger != nil {
		l.logger.DebugContext(ctx, msg, args...)
	}
}

func (l LogAPI) With(args ...any) *slog.Logger {
	if l.logger != nil {
		return l.logger.With(args...)
	}
	return nil
}

func (l LogAPI) WithGroup(name string) *slog.Logger {
	if l.logger != nil {
		return l.logger.WithGroup(name)
	}
	return nil
}
