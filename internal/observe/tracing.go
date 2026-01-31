package observe

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
)

const (
	// TraceHeader is the standard header for request trace IDs.
	TraceHeader = "X-Request-ID"
)

// traceKey is the context key for the trace ID.
type traceKey struct{}

// GenerateTraceID creates a random 16-byte hex string (like a UUID without dashes).
// Uses crypto/rand for uniqueness.
func GenerateTraceID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// TraceIDFromRequest extracts or generates a trace ID for the request.
// If the client sent X-Request-ID, reuse it. Otherwise, generate a new one.
func TraceIDFromRequest(r *http.Request) string {
	if id := r.Header.Get(TraceHeader); id != "" {
		return id
	}
	return GenerateTraceID()
}

// WithTraceID stores the trace ID in the context.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceKey{}, traceID)
}

// TraceIDFrom retrieves the trace ID from context.
func TraceIDFrom(ctx context.Context) string {
	if id, ok := ctx.Value(traceKey{}).(string); ok {
		return id
	}
	return ""
}

// TracingMiddleware is an HTTP middleware that:
//  1. Extracts or generates a trace ID
//  2. Stores it in the request context
//  3. Sets it on the response header
func TracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := TraceIDFromRequest(r)

		// Store in context for downstream use
		ctx := WithTraceID(r.Context(), traceID)
		r = r.WithContext(ctx)

		// Set on outgoing request header (for forwarding to backends)
		r.Header.Set(TraceHeader, traceID)

		// Set on response header (for client)
		w.Header().Set(TraceHeader, traceID)

		next.ServeHTTP(w, r)
	})
}
