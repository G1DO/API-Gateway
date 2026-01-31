package ratelimit

import (
	"sync"
	"time"
)

// TokenBucket implements the token bucket rate limiting algorithm.
//
// Tokens refill lazily: instead of a background ticker, we calculate
// how many tokens to add based on elapsed time when Allow() is called.
type TokenBucket struct {
	mu         sync.Mutex
	tokens     float64   // current tokens (float for fractional refills)
	capacity   float64   // max tokens (= max burst size)
	rate       float64   // tokens added per second (= sustained rate)
	lastRefill time.Time // last time we calculated a refill
}

// NewTokenBucket creates a token bucket that allows bursts up to capacity
// and sustains rate requests per second. Starts full.
func NewTokenBucket(capacity int, rate float64) *TokenBucket {
	return &TokenBucket{
		tokens:     float64(capacity),
		capacity:   float64(capacity),
		rate:       rate,
		lastRefill: time.Now(),
	}
}

// Allow consumes one token and returns true, or returns false if empty.
// When false, retryAfter indicates how long until a token is available.
func (tb *TokenBucket) Allow() (ok bool, retryAfter time.Duration) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.rate
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}
	tb.lastRefill = now

	if tb.tokens >= 1 {
		tb.tokens--
		return true, 0
	}

	// How long until 1 token is available
	deficit := 1 - tb.tokens
	wait := time.Duration(deficit / tb.rate * float64(time.Second))
	return false, wait
}
