package gintelemetry

import (
	"context"
	"log/slog"
	"os"
)

const (
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)

type Level = slog.Level

// applyLevelFilter creates a logger that writes to both OTLP collector and stdout.
// This provides dual output: structured logs to the collector and console output for development.
func applyLevelFilter(otelLogger *slog.Logger, level Level) *slog.Logger {
	// Create stdout handler for console output
	stdoutHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	// Combine OTLP and stdout handlers
	multiHandler := &multiHandler{
		handlers: []slog.Handler{
			otelLogger.Handler(),
			stdoutHandler,
		},
		level: level,
	}

	return slog.New(multiHandler)
}

// multiHandler writes to multiple handlers simultaneously
type multiHandler struct {
	handlers []slog.Handler
	level    Level
}

func (h *multiHandler) Enabled(ctx context.Context, level Level) bool {
	return level >= h.level
}

func (h *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, handler := range h.handlers {
		if err := handler.Handle(ctx, r.Clone()); err != nil {
			// Continue to other handlers even if one fails
			continue
		}
	}
	return nil
}

func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithAttrs(attrs)
	}
	return &multiHandler{handlers: newHandlers, level: h.level}
}

func (h *multiHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithGroup(name)
	}
	return &multiHandler{handlers: newHandlers, level: h.level}
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
