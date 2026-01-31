package health

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// --- Active Health Checks ---

func TestActiveHealthCheckMarksHealthy(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	ac := NewActiveChecker([]string{backend.URL}, Config{
		Interval:           100 * time.Millisecond,
		Timeout:            1 * time.Second,
		HealthPath:         "/",
		HealthyThreshold:   2,
		UnhealthyThreshold: 2,
	})
	defer ac.Close()

	// Wait for a few probes
	time.Sleep(300 * time.Millisecond)

	if !ac.IsHealthy(backend.URL) {
		t.Fatal("backend should be marked healthy after successful probes")
	}
}

func TestActiveHealthCheckMarksUnhealthy(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer backend.Close()

	ac := NewActiveChecker([]string{backend.URL}, Config{
		Interval:           50 * time.Millisecond,
		Timeout:            1 * time.Second,
		HealthPath:         "/",
		HealthyThreshold:   2,
		UnhealthyThreshold: 2,
	})
	defer ac.Close()

	// Wait for consecutive failures
	time.Sleep(200 * time.Millisecond)

	if ac.IsHealthy(backend.URL) {
		t.Fatal("backend should be marked unhealthy after failures")
	}
	if ac.Status(backend.URL) != StatusUnhealthy {
		t.Fatalf("expected unhealthy status, got %s", ac.Status(backend.URL))
	}
}

func TestActiveHealthCheckRecovery(t *testing.T) {
	failing := true
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if failing {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer backend.Close()

	ac := NewActiveChecker([]string{backend.URL}, Config{
		Interval:           50 * time.Millisecond,
		Timeout:            1 * time.Second,
		HealthPath:         "/",
		HealthyThreshold:   2,
		UnhealthyThreshold: 2,
	})
	defer ac.Close()

	// Wait for unhealthy
	time.Sleep(200 * time.Millisecond)
	if ac.IsHealthy(backend.URL) {
		t.Fatal("should be unhealthy")
	}

	// Recover
	failing = false
	time.Sleep(200 * time.Millisecond)

	if !ac.IsHealthy(backend.URL) {
		t.Fatal("backend should recover to healthy")
	}
}

func TestActiveHealthCheckUnreachable(t *testing.T) {
	// Point at non-existent backend
	ac := NewActiveChecker([]string{"http://127.0.0.1:1"}, Config{
		Interval:           50 * time.Millisecond,
		Timeout:            100 * time.Millisecond,
		HealthPath:         "/",
		HealthyThreshold:   2,
		UnhealthyThreshold: 2,
	})
	defer ac.Close()

	time.Sleep(200 * time.Millisecond)

	if ac.IsHealthy("http://127.0.0.1:1") {
		t.Fatal("unreachable backend should be unhealthy")
	}
}

// --- Passive Health Checks ---

func TestPassiveHealthCheckErrorRate(t *testing.T) {
	pc := NewPassiveChecker(PassiveConfig{
		WindowSize:     10 * time.Second,
		ErrorThreshold: 0.5,
		MinRequests:    5,
	})

	backend := "http://backend-A"

	// Record mixed results
	pc.RecordSuccess(backend)
	pc.RecordSuccess(backend)
	pc.RecordFailure(backend)
	pc.RecordFailure(backend)
	pc.RecordFailure(backend)

	// 3/5 = 60% error rate, threshold is 50%
	if pc.IsHealthy(backend) {
		t.Fatal("backend should be unhealthy (60% error rate)")
	}
	if rate := pc.ErrorRate(backend); rate < 0.59 || rate > 0.61 {
		t.Fatalf("expected ~0.6 error rate, got %.2f", rate)
	}
}

func TestPassiveHealthCheckMinRequests(t *testing.T) {
	pc := NewPassiveChecker(PassiveConfig{
		WindowSize:     10 * time.Second,
		ErrorThreshold: 0.5,
		MinRequests:    10,
	})

	backend := "http://backend-B"

	// Only 3 requests (below minRequests)
	pc.RecordFailure(backend)
	pc.RecordFailure(backend)
	pc.RecordFailure(backend)

	// Should still be healthy (not enough data)
	if !pc.IsHealthy(backend) {
		t.Fatal("should be healthy when below minRequests")
	}
}

func TestPassiveHealthCheckSlidingWindow(t *testing.T) {
	pc := NewPassiveChecker(PassiveConfig{
		WindowSize:     100 * time.Millisecond,
		ErrorThreshold: 0.5,
		MinRequests:    3,
	})

	backend := "http://backend-C"

	// Record failures
	pc.RecordFailure(backend)
	pc.RecordFailure(backend)
	pc.RecordFailure(backend)

	if pc.IsHealthy(backend) {
		t.Fatal("should be unhealthy")
	}

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Old failures expired, should be healthy (no recent data)
	if !pc.IsHealthy(backend) {
		t.Fatal("should be healthy after window expires")
	}
}

// --- Combined Checker ---

func TestCombinedCheckerBothPass(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	active := NewActiveChecker([]string{backend.URL}, Config{
		Interval:           50 * time.Millisecond,
		Timeout:            1 * time.Second,
		HealthPath:         "/",
		HealthyThreshold:   2,
		UnhealthyThreshold: 2,
	})
	defer active.Close()

	passive := NewPassiveChecker(PassiveConfig{
		WindowSize:     10 * time.Second,
		ErrorThreshold: 0.5,
		MinRequests:    3,
	})

	combined := NewCombined(active, passive)

	// Wait for active checks
	time.Sleep(150 * time.Millisecond)

	// Record successful passive checks
	combined.RecordSuccess(backend.URL)
	combined.RecordSuccess(backend.URL)
	combined.RecordSuccess(backend.URL)

	if !combined.IsHealthy(backend.URL) {
		t.Fatal("should be healthy when both pass")
	}
}

func TestCombinedCheckerEitherFails(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	active := NewActiveChecker([]string{backend.URL}, Config{
		Interval:           50 * time.Millisecond,
		Timeout:            1 * time.Second,
		HealthPath:         "/",
		HealthyThreshold:   2,
		UnhealthyThreshold: 2,
	})
	defer active.Close()

	passive := NewPassiveChecker(PassiveConfig{
		WindowSize:     10 * time.Second,
		ErrorThreshold: 0.5,
		MinRequests:    3,
	})

	combined := NewCombined(active, passive)

	time.Sleep(150 * time.Millisecond)

	// Active is healthy, but passive has high error rate
	combined.RecordFailure(backend.URL)
	combined.RecordFailure(backend.URL)
	combined.RecordFailure(backend.URL)

	if combined.IsHealthy(backend.URL) {
		t.Fatal("should be unhealthy when passive fails")
	}
}

// --- Healthy Pool ---

func TestHealthyPoolFiltersUnhealthy(t *testing.T) {
	healthyBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer healthyBackend.Close()

	unhealthyBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer unhealthyBackend.Close()

	backends := []string{healthyBackend.URL, unhealthyBackend.URL}

	active := NewActiveChecker(backends, Config{
		Interval:           50 * time.Millisecond,
		Timeout:            1 * time.Second,
		HealthPath:         "/",
		HealthyThreshold:   2,
		UnhealthyThreshold: 2,
	})
	defer active.Close()

	passive := NewPassiveChecker(PassiveConfig{
		WindowSize:     10 * time.Second,
		ErrorThreshold: 0.5,
		MinRequests:    100, // high threshold so passive doesn't interfere
	})

	combined := NewCombined(active, passive)
	pool := NewHealthyPool(backends, combined)

	// Wait for health checks
	time.Sleep(200 * time.Millisecond)

	healthy := pool.Healthy()
	if len(healthy) != 1 {
		t.Fatalf("expected 1 healthy backend, got %d", len(healthy))
	}
	if healthy[0] != healthyBackend.URL {
		t.Fatal("wrong backend marked healthy")
	}
}

func TestHealthyPoolAllUnhealthyFailOpen(t *testing.T) {
	backends := []string{"http://127.0.0.1:1", "http://127.0.0.1:2"}

	active := NewActiveChecker(backends, Config{
		Interval:           50 * time.Millisecond,
		Timeout:            100 * time.Millisecond,
		HealthPath:         "/",
		HealthyThreshold:   2,
		UnhealthyThreshold: 2,
	})
	defer active.Close()

	passive := NewPassiveChecker(PassiveConfig{
		WindowSize:     10 * time.Second,
		ErrorThreshold: 0.5,
		MinRequests:    100,
	})

	combined := NewCombined(active, passive)
	pool := NewHealthyPool(backends, combined)

	time.Sleep(200 * time.Millisecond)

	// All unhealthy → fail-open returns all
	healthy := pool.Healthy()
	if len(healthy) != 2 {
		t.Fatalf("expected fail-open to return all backends, got %d", len(healthy))
	}
}

func TestHealthyPoolAllUnhealthyFailClosed(t *testing.T) {
	backends := []string{"http://127.0.0.1:1"}

	active := NewActiveChecker(backends, Config{
		Interval:           50 * time.Millisecond,
		Timeout:            100 * time.Millisecond,
		HealthPath:         "/",
		HealthyThreshold:   2,
		UnhealthyThreshold: 2,
	})
	defer active.Close()

	passive := NewPassiveChecker(PassiveConfig{
		WindowSize:     10 * time.Second,
		ErrorThreshold: 0.5,
		MinRequests:    100,
	})

	combined := NewCombined(active, passive)
	pool := NewHealthyPool(backends, combined)

	time.Sleep(200 * time.Millisecond)

	_, err := pool.HealthyOrError()
	if err != ErrAllBackendsUnhealthy {
		t.Fatalf("expected ErrAllBackendsUnhealthy, got %v", err)
	}
}