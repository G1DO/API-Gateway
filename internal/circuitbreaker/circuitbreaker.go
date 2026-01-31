package circuitbreaker

import (
	"sync"
	"sync/atomic"
	"time"
)

// State represents circuit breaker states.
type State uint32

const (
	StateClosed   State = iota // Normal: requests pass through
	StateOpen                   // Tripped: reject all requests immediately
	StateHalfOpen               // Testing: allow one request to test recovery
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker implements the circuit breaker pattern.
//
// State transitions:
//   Closed → Open:      after maxFailures consecutive failures
//   Open → Half-Open:   after timeout duration
//   Half-Open → Closed: after one successful request
//   Half-Open → Open:   after one failed request
type CircuitBreaker struct {
	maxFailures int
	timeout     time.Duration

	mu              sync.Mutex
	state           atomic.Uint32 // State (for fast reads without lock)
	failures        int
	lastFailureTime time.Time
}

// New creates a circuit breaker that opens after maxFailures consecutive
// failures and transitions to half-open after timeout.
func New(maxFailures int, timeout time.Duration) *CircuitBreaker {
	cb := &CircuitBreaker{
		maxFailures: maxFailures,
		timeout:     timeout,
	}
	cb.state.Store(uint32(StateClosed))
	return cb
}

// Allow returns true if the request should proceed.
// Returns false when circuit is open.
func (cb *CircuitBreaker) Allow() bool {
	state := State(cb.state.Load())

	switch state {
	case StateClosed:
		return true

	case StateOpen:
		// Check if timeout has passed → transition to half-open
		cb.mu.Lock()
		if time.Since(cb.lastFailureTime) >= cb.timeout {
			cb.setState(StateHalfOpen)
			cb.mu.Unlock()
			return true // allow the test request
		}
		cb.mu.Unlock()
		return false

	case StateHalfOpen:
		// Only the first caller gets through; others are rejected
		// until the test request completes (success or failure)
		return false

	default:
		return false
	}
}

// RecordSuccess resets the failure count and closes the circuit if half-open.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	if State(cb.state.Load()) == StateHalfOpen {
		cb.setState(StateClosed)
	}
}

// RecordFailure increments the failure count and opens the circuit
// if maxFailures is reached.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailureTime = time.Now()

	state := State(cb.state.Load())

	if state == StateHalfOpen {
		// Test request failed → reopen
		cb.setState(StateOpen)
		return
	}

	if cb.failures >= cb.maxFailures {
		cb.setState(StateOpen)
	}
}

// State returns the current circuit breaker state.
func (cb *CircuitBreaker) State() State {
	return State(cb.state.Load())
}

// setState updates the state (must hold mu).
func (cb *CircuitBreaker) setState(s State) {
	cb.state.Store(uint32(s))
}
