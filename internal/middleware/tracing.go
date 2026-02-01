package middleware

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
)

const traceHeader = "X-Request-ID"

type traceKey struct{}

// Tracing generates or propagates a trace ID for each request.
// If the client sends X-Request-ID, it's reused. Otherwise a new one is generated.
// The trace ID is stored in the context and set on the response header.
func Tracing() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			traceID := r.Header.Get(traceHeader)
			if traceID == "" {
				b := make([]byte, 16)
				rand.Read(b)
				traceID = fmt.Sprintf("%x", b)
			}

			ctx := context.WithValue(r.Context(), traceKey{}, traceID)
			r = r.WithContext(ctx)
			r.Header.Set(traceHeader, traceID)
			w.Header().Set(traceHeader, traceID)

			next.ServeHTTP(w, r)
		})
	}
}

// TraceIDFrom retrieves the trace ID from context.
func TraceIDFrom(ctx context.Context) string {
	if id, ok := ctx.Value(traceKey{}).(string); ok {
		return id
	}
	return ""
}
