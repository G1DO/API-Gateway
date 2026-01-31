package circuitbreaker

import (
	"sync"
	"time"
)

// PerBackend maintains a separate circuit breaker for each backend address.
//
// This ensures that one failing backend doesn't cause the gateway to
// reject requests to healthy backends.
type PerBackend struct {
	mu          sync.RWMutex
	breakers    map[string]*CircuitBreaker
	maxFailures int
	timeout     time.Duration
}

// NewPerBackend creates a per-backend circuit breaker manager.
// Each backend gets a circuit that opens after maxFailures consecutive
// failures and transitions to half-open after timeout.
func NewPerBackend(maxFailures int, timeout time.Duration) *PerBackend {
	return &PerBackend{
		breakers:    make(map[string]*CircuitBreaker),
		maxFailures: maxFailures,
		timeout:     timeout,
	}
}

// Allow checks if requests to the given backend are allowed.
func (pb *PerBackend) Allow(backend string) bool {
	cb := pb.get(backend)
	return cb.Allow()
}

// RecordSuccess records a successful request to the backend.
func (pb *PerBackend) RecordSuccess(backend string) {
	cb := pb.get(backend)
	cb.RecordSuccess()
}

// RecordFailure records a failed request to the backend.
func (pb *PerBackend) RecordFailure(backend string) {
	cb := pb.get(backend)
	cb.RecordFailure()
}

// State returns the current state of the circuit for the given backend.
func (pb *PerBackend) State(backend string) State {
	cb := pb.get(backend)
	return cb.State()
}

// get returns the circuit breaker for a backend, creating it lazily if needed.
func (pb *PerBackend) get(backend string) *CircuitBreaker {
	// Fast path: breaker already exists
	pb.mu.RLock()
	cb, exists := pb.breakers[backend]
	pb.mu.RUnlock()
	if exists {
		return cb
	}

	// Slow path: create breaker
	pb.mu.Lock()
	defer pb.mu.Unlock()

	// Double-check after acquiring write lock
	cb, exists = pb.breakers[backend]
	if exists {
		return cb
	}

	cb = New(pb.maxFailures, pb.timeout)
	pb.breakers[backend] = cb
	return cb
}
