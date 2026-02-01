package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/G1D0/Api-Gateway/internal/ratelimit"
)

// RateLimit rejects requests with 429 when the client exceeds their rate limit.
// Uses per-client token bucket rate limiting.
func RateLimit(limiter *ratelimit.PerClient) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := r.RemoteAddr

			ok, retryAfter := limiter.Allow(clientIP)
			if !ok {
				w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
				http.Error(w, "rate limited", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitWithKeyFunc is like RateLimit but uses a custom function to extract
// the client key (e.g., API key from header instead of IP).
func RateLimitWithKeyFunc(limiter *ratelimit.PerClient, keyFunc func(*http.Request) string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)

			ok, retryAfter := limiter.Allow(key)
			if !ok {
				w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
				http.Error(w, "rate limited", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// NewDefaultLimiter creates a per-client rate limiter with sensible defaults.
func NewDefaultLimiter() *ratelimit.PerClient {
	return ratelimit.NewPerClient(
		100,              // 100 burst
		10.0,             // 10 req/sec sustained
		10*time.Minute,   // stale bucket cleanup
	)
}
