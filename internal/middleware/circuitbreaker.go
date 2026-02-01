package middleware

import (
	"net/http"

	"github.com/G1D0/Api-Gateway/internal/circuitbreaker"
)

// CircuitBreaker rejects requests with 503 when the backend's circuit is open.
// Records success/failure after the request completes.
func CircuitBreaker(cb *circuitbreaker.PerBackend, backendFunc func(*http.Request) string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			backend := backendFunc(r)

			if !cb.Allow(backend) {
				http.Error(w, "service unavailable", http.StatusServiceUnavailable)
				return
			}

			rc := NewResponseCapture(w)
			next.ServeHTTP(rc, r)

			// Record outcome based on response status
			if rc.StatusCode >= 500 {
				cb.RecordFailure(backend)
			} else {
				cb.RecordSuccess(backend)
			}
		})
	}
}
