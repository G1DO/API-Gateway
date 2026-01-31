package circuitbreaker

import (
	"sync"
	"testing"
	"time"
)

// --- Circuit Breaker State Machine ---

func TestCircuitBreakerClosed(t *testing.T) {
	cb := New(3, 100*time.Millisecond)

	if cb.State() != StateClosed {
		t.Fatal("should start closed")
	}
	if !cb.Allow() {
		t.Fatal("should allow requests when closed")
	}
}

func TestCircuitBreakerOpensAfterFailures(t *testing.T) {
	cb := New(3, 100*time.Millisecond)

	// 3 failures should open the circuit
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.State() != StateClosed {
		t.Fatal("should still be closed after 2 failures")
	}

	cb.RecordFailure()
	if cb.State() != StateOpen {
		t.Fatal("should be open after 3 failures")
	}
	if cb.Allow() {
		t.Fatal("should reject requests when open")
	}
}

func TestCircuitBreakerResetsOnSuccess(t *testing.T) {
	cb := New(3, 100*time.Millisecond)

	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordSuccess() // reset

	if cb.State() != StateClosed {
		t.Fatal("should remain closed after success")
	}

	// Now need 3 more failures to open
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.State() != StateClosed {
		t.Fatal("counter should have reset")
	}
}

func TestCircuitBreakerHalfOpen(t *testing.T) {
	cb := New(2, 50*time.Millisecond)

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.State() != StateOpen {
		t.Fatal("should be open")
	}

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)

	// First Allow after timeout should transition to half-open and return true
	if !cb.Allow() {
		t.Fatal("first request after timeout should be allowed (half-open)")
	}
	if cb.State() != StateHalfOpen {
		t.Fatalf("should be half-open, got %s", cb.State())
	}

	// Subsequent requests should be rejected (only one test request allowed)
	if cb.Allow() {
		t.Fatal("should reject additional requests in half-open")
	}
}

func TestCircuitBreakerHalfOpenToClosedOnSuccess(t *testing.T) {
	cb := New(2, 50*time.Millisecond)

	cb.RecordFailure()
	cb.RecordFailure()
	time.Sleep(100 * time.Millisecond)
	cb.Allow() // transition to half-open

	cb.RecordSuccess() // test request succeeded
	if cb.State() != StateClosed {
		t.Fatal("should close after successful test request")
	}
	if !cb.Allow() {
		t.Fatal("should allow requests after closing")
	}
}

func TestCircuitBreakerHalfOpenToOpenOnFailure(t *testing.T) {
	cb := New(2, 50*time.Millisecond)

	cb.RecordFailure()
	cb.RecordFailure()
	time.Sleep(100 * time.Millisecond)
	cb.Allow() // transition to half-open

	cb.RecordFailure() // test request failed
	if cb.State() != StateOpen {
		t.Fatal("should reopen after failed test request")
	}
	if cb.Allow() {
		t.Fatal("should reject requests after reopening")
	}
}

func TestCircuitBreakerConcurrent(t *testing.T) {
	cb := New(10, 100*time.Millisecond)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cb.Allow()
			if i%2 == 0 {
				cb.RecordSuccess()
			} else {
				cb.RecordFailure()
			}
		}()
	}
	wg.Wait()
}

// --- Per-Backend Circuits ---

func TestPerBackendIsolation(t *testing.T) {
	pb := NewPerBackend(2, 100*time.Millisecond)

	// Fail backend A
	pb.RecordFailure("A")
	pb.RecordFailure("A")

	if pb.State("A") != StateOpen {
		t.Fatal("A should be open")
	}
	if pb.State("B") != StateClosed {
		t.Fatal("B should be closed (unaffected)")
	}

	if pb.Allow("A") {
		t.Fatal("A should reject")
	}
	if !pb.Allow("B") {
		t.Fatal("B should allow")
	}
}

func TestPerBackendLazyCreation(t *testing.T) {
	pb := NewPerBackend(3, 100*time.Millisecond)

	// First request to new backend should be allowed
	if !pb.Allow("new-backend") {
		t.Fatal("first request should be allowed (circuit starts closed)")
	}
}

func TestPerBackendConcurrent(t *testing.T) {
	pb := NewPerBackend(5, 100*time.Millisecond)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			backend := "backend-A"
			if n%2 == 0 {
				backend = "backend-B"
			}
			pb.Allow(backend)
			if n%3 == 0 {
				pb.RecordSuccess(backend)
			} else {
				pb.RecordFailure(backend)
			}
		}(i)
	}
	wg.Wait()
}

func TestPerBackendRecovery(t *testing.T) {
	pb := NewPerBackend(2, 50*time.Millisecond)

	// Open circuit
	pb.RecordFailure("X")
	pb.RecordFailure("X")
	if pb.State("X") != StateOpen {
		t.Fatal("should be open")
	}

	// Wait for half-open transition
	time.Sleep(100 * time.Millisecond)
	pb.Allow("X") // transition to half-open

	// Successful test request closes circuit
	pb.RecordSuccess("X")
	if pb.State("X") != StateClosed {
		t.Fatal("should be closed after recovery")
	}
}