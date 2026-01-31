package health

import (
	"sync"
	"time"
)

// requestOutcome tracks a single request result.
type requestOutcome struct {
	timestamp time.Time
	success   bool
}

// passiveBackend tracks passive health metrics for one backend.
type passiveBackend struct {
	mu       sync.Mutex
	outcomes []requestOutcome
}

// PassiveChecker infers backend health from real traffic patterns.
type PassiveChecker struct {
	mu       sync.RWMutex
	backends map[string]*passiveBackend

	windowSize       time.Duration // how far back to look
	errorThreshold   float64       // e.g., 0.5 = 50% error rate triggers unhealthy
	minRequests      int           // minimum requests in window before judging
}

// PassiveConfig holds passive health check configuration.
type PassiveConfig struct {
	WindowSize     time.Duration // e.g., 30s
	ErrorThreshold float64       // e.g., 0.5 (50%)
	MinRequests    int           // e.g., 10
}

// NewPassiveChecker creates a passive health checker.
func NewPassiveChecker(cfg PassiveConfig) *PassiveChecker {
	return &PassiveChecker{
		backends:       make(map[string]*passiveBackend),
		windowSize:     cfg.WindowSize,
		errorThreshold: cfg.ErrorThreshold,
		minRequests:    cfg.MinRequests,
	}
}

// RecordSuccess records a successful request to a backend.
func (pc *PassiveChecker) RecordSuccess(backend string) {
	pc.record(backend, true)
}

// RecordFailure records a failed request to a backend.
func (pc *PassiveChecker) RecordFailure(backend string) {
	pc.record(backend, false)
}

// record adds an outcome to the sliding window.
func (pc *PassiveChecker) record(backend string, success bool) {
	pb := pc.getOrCreate(backend)

	pb.mu.Lock()
	defer pb.mu.Unlock()

	now := time.Now()
	pb.outcomes = append(pb.outcomes, requestOutcome{
		timestamp: now,
		success:   success,
	})

	// Trim old outcomes outside the window
	cutoff := now.Add(-pc.windowSize)
	i := 0
	for i < len(pb.outcomes) && pb.outcomes[i].timestamp.Before(cutoff) {
		i++
	}
	pb.outcomes = pb.outcomes[i:]
}

// IsHealthy returns true if the backend's error rate is below threshold.
func (pc *PassiveChecker) IsHealthy(backend string) bool {
	pc.mu.RLock()
	pb, exists := pc.backends[backend]
	pc.mu.RUnlock()

	if !exists {
		return true // no data = assume healthy
	}

	pb.mu.Lock()
	defer pb.mu.Unlock()

	// Clean stale data
	now := time.Now()
	cutoff := now.Add(-pc.windowSize)
	i := 0
	for i < len(pb.outcomes) && pb.outcomes[i].timestamp.Before(cutoff) {
		i++
	}
	pb.outcomes = pb.outcomes[i:]

	if len(pb.outcomes) < pc.minRequests {
		return true // not enough data
	}

	failures := 0
	for _, outcome := range pb.outcomes {
		if !outcome.success {
			failures++
		}
	}

	errorRate := float64(failures) / float64(len(pb.outcomes))
	return errorRate < pc.errorThreshold
}

// ErrorRate returns the current error rate for a backend (for monitoring).
func (pc *PassiveChecker) ErrorRate(backend string) float64 {
	pc.mu.RLock()
	pb, exists := pc.backends[backend]
	pc.mu.RUnlock()

	if !exists {
		return 0
	}

	pb.mu.Lock()
	defer pb.mu.Unlock()

	if len(pb.outcomes) == 0 {
		return 0
	}

	failures := 0
	for _, outcome := range pb.outcomes {
		if !outcome.success {
			failures++
		}
	}
	return float64(failures) / float64(len(pb.outcomes))
}

// getOrCreate returns the passive backend, creating it if needed.
func (pc *PassiveChecker) getOrCreate(backend string) *passiveBackend {
	pc.mu.RLock()
	pb, exists := pc.backends[backend]
	pc.mu.RUnlock()
	if exists {
		return pb
	}

	pc.mu.Lock()
	defer pc.mu.Unlock()

	// Double-check
	pb, exists = pc.backends[backend]
	if exists {
		return pb
	}

	pb = &passiveBackend{}
	pc.backends[backend] = pb
	return pb
}