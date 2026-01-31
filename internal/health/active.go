package health

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// Status represents backend health status.
type Status int

const (
	StatusUnknown Status = iota
	StatusHealthy
	StatusUnhealthy
)

func (s Status) String() string {
	switch s {
	case StatusHealthy:
		return "healthy"
	case StatusUnhealthy:
		return "unhealthy"
	default:
		return "unknown"
	}
}

// backendStatus tracks health state for a single backend.
type backendStatus struct {
	mu                sync.RWMutex
	status            Status
	consecutiveSuccesses int
	consecutiveFailures  int
}

// ActiveChecker periodically probes backends with health check requests.
type ActiveChecker struct {
	mu       sync.RWMutex
	backends map[string]*backendStatus

	interval            time.Duration
	timeout             time.Duration
	healthPath          string
	healthyThreshold    int // consecutive successes to mark healthy
	unhealthyThreshold  int // consecutive failures to mark unhealthy

	client *http.Client
	ctx    context.Context
	cancel context.CancelFunc
}

// Config holds active health check configuration.
type Config struct {
	Interval           time.Duration // how often to probe
	Timeout            time.Duration // per-probe timeout
	HealthPath         string        // e.g., "/health"
	HealthyThreshold   int           // consecutive successes
	UnhealthyThreshold int           // consecutive failures
}

// NewActiveChecker creates and starts an active health checker.
func NewActiveChecker(backends []string, cfg Config) *ActiveChecker {
	ctx, cancel := context.WithCancel(context.Background())

	ac := &ActiveChecker{
		backends:           make(map[string]*backendStatus),
		interval:           cfg.Interval,
		timeout:            cfg.Timeout,
		healthPath:         cfg.HealthPath,
		healthyThreshold:   cfg.HealthyThreshold,
		unhealthyThreshold: cfg.UnhealthyThreshold,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize backends as unknown
	for _, addr := range backends {
		ac.backends[addr] = &backendStatus{
			status: StatusUnknown,
		}
	}

	go ac.run()
	return ac
}

// IsHealthy returns true if the backend is healthy.
func (ac *ActiveChecker) IsHealthy(backend string) bool {
	ac.mu.RLock()
	bs, exists := ac.backends[backend]
	ac.mu.RUnlock()

	if !exists {
		return true // optimistic: unknown backends are assumed healthy
	}

	bs.mu.RLock()
	defer bs.mu.RUnlock()
	return bs.status == StatusHealthy || bs.status == StatusUnknown
}

// Status returns the current health status of a backend.
func (ac *ActiveChecker) Status(backend string) Status {
	ac.mu.RLock()
	bs, exists := ac.backends[backend]
	ac.mu.RUnlock()

	if !exists {
		return StatusUnknown
	}

	bs.mu.RLock()
	defer bs.mu.RUnlock()
	return bs.status
}

// Close stops the health checker.
func (ac *ActiveChecker) Close() {
	ac.cancel()
}

// run is the background goroutine that probes backends.
func (ac *ActiveChecker) run() {
	ticker := time.NewTicker(ac.interval)
	defer ticker.Stop()

	// Probe immediately on startup
	ac.probeAll()

	for {
		select {
		case <-ticker.C:
			ac.probeAll()
		case <-ac.ctx.Done():
			return
		}
	}
}

// probeAll checks all backends concurrently.
func (ac *ActiveChecker) probeAll() {
	ac.mu.RLock()
	backends := make([]string, 0, len(ac.backends))
	for addr := range ac.backends {
		backends = append(backends, addr)
	}
	ac.mu.RUnlock()

	var wg sync.WaitGroup
	for _, addr := range backends {
		wg.Add(1)
		go func(backend string) {
			defer wg.Done()
			ac.probe(backend)
		}(addr)
	}
	wg.Wait()
}

// probe sends a health check request to one backend.
func (ac *ActiveChecker) probe(backend string) {
	url := backend + ac.healthPath

	req, err := http.NewRequestWithContext(ac.ctx, http.MethodGet, url, nil)
	if err != nil {
		ac.recordFailure(backend)
		return
	}

	resp, err := ac.client.Do(req)
	if err != nil {
		ac.recordFailure(backend)
		return
	}
	defer resp.Body.Close()

	// Consider 2xx as healthy
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		ac.recordSuccess(backend)
	} else {
		ac.recordFailure(backend)
	}
}

// recordSuccess updates state after a successful health check.
func (ac *ActiveChecker) recordSuccess(backend string) {
	ac.mu.RLock()
	bs := ac.backends[backend]
	ac.mu.RUnlock()

	bs.mu.Lock()
	defer bs.mu.Unlock()

	bs.consecutiveSuccesses++
	bs.consecutiveFailures = 0

	if bs.consecutiveSuccesses >= ac.healthyThreshold {
		bs.status = StatusHealthy
	}
}

// recordFailure updates state after a failed health check.
func (ac *ActiveChecker) recordFailure(backend string) {
	ac.mu.RLock()
	bs := ac.backends[backend]
	ac.mu.RUnlock()

	bs.mu.Lock()
	defer bs.mu.Unlock()

	bs.consecutiveFailures++
	bs.consecutiveSuccesses = 0

	if bs.consecutiveFailures >= ac.unhealthyThreshold {
		bs.status = StatusUnhealthy
	}
}

// AddBackend dynamically adds a new backend to monitor.
func (ac *ActiveChecker) AddBackend(backend string) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if _, exists := ac.backends[backend]; exists {
		return
	}

	ac.backends[backend] = &backendStatus{
		status: StatusUnknown,
	}
}

// RemoveBackend stops monitoring a backend.
func (ac *ActiveChecker) RemoveBackend(backend string) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	delete(ac.backends, backend)
}

// AllStatus returns a snapshot of all backend statuses (for debugging/monitoring).
func (ac *ActiveChecker) AllStatus() map[string]Status {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	result := make(map[string]Status, len(ac.backends))
	for addr, bs := range ac.backends {
		bs.mu.RLock()
		result[addr] = bs.status
		bs.mu.RUnlock()
	}
	return result
}