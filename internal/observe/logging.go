package observe

import (
	"context"
	"log/slog"
	"os"
)

// Level aliases for convenience.
const (
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)

// loggerKey is the context key for the request-scoped logger.
type loggerKey struct{}

// NewLogger creates a structured JSON logger with the given minimum level.
func NewLogger(level slog.Level) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
}

// WithLogger stores a logger in the context.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// LoggerFrom retrieves the logger from context, or returns the default logger.
func LoggerFrom(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}

// RequestLogger creates a logger with request-scoped fields pre-attached.
// All subsequent log calls include these fields automatically.
func RequestLogger(base *slog.Logger, method, path, clientIP, traceID string) *slog.Logger {
	return base.With(
		"method", method,
		"path", path,
		"client_ip", clientIP,
		"trace_id", traceID,
	)
}
