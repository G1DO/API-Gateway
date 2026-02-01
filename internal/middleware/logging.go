package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// Logging logs each request as structured JSON with method, path, status,
// latency, client IP, and trace ID.
func Logging(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rc := NewResponseCapture(w)

			next.ServeHTTP(rc, r)

			logger.Info("request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rc.StatusCode,
				"latency_ms", time.Since(start).Milliseconds(),
				"client_ip", r.RemoteAddr,
				"trace_id", TraceIDFrom(r.Context()),
			)
		})
	}
}
