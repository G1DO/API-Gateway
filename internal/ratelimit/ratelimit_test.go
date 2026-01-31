package ratelimit

import (
	"sync"
	"testing"
	"time"
)

// --- Token Bucket ---

func TestTokenBucketAllowsBurst(t *testing.T) {
	tb := NewTokenBucket(5, 1.0) // 5 burst, 1/sec sustained

	for i := 0; i < 5; i++ {
		ok, _ := tb.Allow()
		if !ok {
			t.Fatalf("request %d should be allowed (burst)", i)
		}
	}

	// 6th should be rejected
	ok, retry := tb.Allow()
	if ok {
		t.Fatal("6th request should be rejected")
	}
	if retry <= 0 {
		t.Fatal("retry-after should be positive")
	}
}

func TestTokenBucketRefills(t *testing.T) {
	tb := NewTokenBucket(2, 10.0) // 2 burst, 10/sec refill

	// Drain the bucket
	tb.Allow()
	tb.Allow()
	ok, _ := tb.Allow()
	if ok {
		t.Fatal("should be empty")
	}

	// Wait for refill (at 10/sec, 1 token in 100ms)
	time.Sleep(150 * time.Millisecond)

	ok, _ = tb.Allow()
	if !ok {
		t.Fatal("should have refilled at least 1 token")
	}
}

func TestTokenBucketDoesNotExceedCapacity(t *testing.T) {
	tb := NewTokenBucket(3, 100.0) // high refill rate

	time.Sleep(100 * time.Millisecond) // way more than enough to refill

	// Should only allow 3 (capacity), not more
	allowed := 0
	for i := 0; i < 10; i++ {
		ok, _ := tb.Allow()
		if ok {
			allowed++
		}
	}
	if allowed != 3 {
		t.Fatalf("expected 3 allowed (capacity), got %d", allowed)
	}
}

func TestTokenBucketConcurrent(t *testing.T) {
	tb := NewTokenBucket(100, 0) // 100 tokens, no refill

	var wg sync.WaitGroup
	allowed := make(chan bool, 200)

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, _ := tb.Allow()
			allowed <- ok
		}()
	}
	wg.Wait()
	close(allowed)

	count := 0
	for ok := range allowed {
		if ok {
			count++
		}
	}
	if count != 100 {
		t.Fatalf("expected exactly 100 allowed, got %d", count)
	}
}

// --- Per-Client ---

func TestPerClientIsolation(t *testing.T) {
	pc := NewPerClient(2, 0, 10*time.Minute) // 2 tokens, no refill
	defer pc.Close()

	// Client A uses all tokens
	pc.Allow("A")
	pc.Allow("A")
	ok, _ := pc.Allow("A")
	if ok {
		t.Fatal("A should be rate limited")
	}

	// Client B should still have tokens
	ok, _ = pc.Allow("B")
	if !ok {
		t.Fatal("B should not be affected by A")
	}
}

func TestPerClientCreatesOnDemand(t *testing.T) {
	pc := NewPerClient(5, 1.0, 10*time.Minute)
	defer pc.Close()

	// First request from new client should succeed
	ok, _ := pc.Allow("new-client")
	if !ok {
		t.Fatal("first request from new client should be allowed")
	}
}

func TestPerClientGarbageCollection(t *testing.T) {
	stale := 100 * time.Millisecond
	pc := NewPerClient(5, 1.0, stale)
	defer pc.Close()

	pc.Allow("temp-client")

	// Wait for GC to run (threshold/2 = 50ms, plus some margin)
	time.Sleep(250 * time.Millisecond)

	pc.mu.RLock()
	_, exists := pc.clients["temp-client"]
	pc.mu.RUnlock()

	if exists {
		t.Fatal("stale client should have been garbage collected")
	}
}

func TestPerClientConcurrent(t *testing.T) {
	pc := NewPerClient(1000, 0, 10*time.Minute)
	defer pc.Close()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pc.Allow("shared-key")
		}()
	}
	wg.Wait()
}

// --- Sliding Window ---

func TestSlidingWindowBasic(t *testing.T) {
	sw := NewSlidingWindow(5, 1*time.Second) // 5 req per second

	for i := 0; i < 5; i++ {
		ok, _ := sw.Allow()
		if !ok {
			t.Fatalf("request %d should be allowed", i)
		}
	}

	ok, retry := sw.Allow()
	if ok {
		t.Fatal("6th request should be rejected")
	}
	if retry <= 0 {
		t.Fatal("retry-after should be positive")
	}
}

func TestSlidingWindowResetsAfterWindow(t *testing.T) {
	sw := NewSlidingWindow(2, 100*time.Millisecond)

	sw.Allow()
	sw.Allow()
	ok, _ := sw.Allow()
	if ok {
		t.Fatal("should be limited")
	}

	// Wait for window to pass
	time.Sleep(200 * time.Millisecond)

	ok, _ = sw.Allow()
	if !ok {
		t.Fatal("should be allowed after window reset")
	}
}

func TestSlidingWindowWeightsPreviousWindow(t *testing.T) {
	sw := NewSlidingWindow(10, 100*time.Millisecond)

	// Fill up previous window
	for i := 0; i < 10; i++ {
		sw.Allow()
	}

	// Move into the next window (just barely)
	time.Sleep(110 * time.Millisecond)

	// Previous window had 10 requests. We're ~10% into the new window,
	// so ~90% of previous window still counts. Effective ≈ 9 + current.
	// We should only be able to make ~1 request before hitting the limit.
	allowed := 0
	for i := 0; i < 5; i++ {
		ok, _ := sw.Allow()
		if ok {
			allowed++
		}
	}

	// Should allow very few (1-2) requests
	if allowed > 3 {
		t.Fatalf("expected ≤3 allowed (previous window weight), got %d", allowed)
	}
}

func TestSlidingWindowConcurrent(t *testing.T) {
	sw := NewSlidingWindow(100, 1*time.Second)

	var wg sync.WaitGroup
	allowed := make(chan bool, 200)

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, _ := sw.Allow()
			allowed <- ok
		}()
	}
	wg.Wait()
	close(allowed)

	count := 0
	for ok := range allowed {
		if ok {
			count++
		}
	}
	if count != 100 {
		t.Fatalf("expected 100 allowed, got %d", count)
	}
}