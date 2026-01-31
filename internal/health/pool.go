package health

import (
	"errors"
	"sync"
)

var (
	// ErrAllBackendsUnhealthy is returned when all backends are unhealthy.
	ErrAllBackendsUnhealthy = errors.New("all backends are unhealthy")
)

// HealthyPool manages a pool of backends, filtering out unhealthy ones.
type HealthyPool struct {
	mu       sync.RWMutex
	all      []string          // all configured backends
	checker  *CombinedChecker
}

// NewHealthyPool creates a pool that filters backends based on health checks.
func NewHealthyPool(backends []string, checker *CombinedChecker) *HealthyPool {
	return &HealthyPool{
		all:     backends,
		checker: checker,
	}
}

// Healthy returns a slice of currently healthy backends.
// Returns all backends if all are unhealthy (fail-open strategy).
func (hp *HealthyPool) Healthy() []string {
	hp.mu.RLock()
	defer hp.mu.RUnlock()

	healthy := make([]string, 0, len(hp.all))
	for _, backend := range hp.all {
		if hp.checker.IsHealthy(backend) {
			healthy = append(healthy, backend)
		}
	}

	// Fail-open: if all unhealthy, return all (maybe health checks are wrong)
	if len(healthy) == 0 {
		return append([]string(nil), hp.all...) // return copy
	}

	return healthy
}

// HealthyOrError returns healthy backends or an error if none are healthy.
// Use this if you prefer fail-closed (return error) instead of fail-open.
func (hp *HealthyPool) HealthyOrError() ([]string, error) {
	hp.mu.RLock()
	defer hp.mu.RUnlock()

	healthy := make([]string, 0, len(hp.all))
	for _, backend := range hp.all {
		if hp.checker.IsHealthy(backend) {
			healthy = append(healthy, backend)
		}
	}

	if len(healthy) == 0 {
		return nil, ErrAllBackendsUnhealthy
	}

	return healthy, nil
}

// All returns all backends regardless of health.
func (hp *HealthyPool) All() []string {
	hp.mu.RLock()
	defer hp.mu.RUnlock()
	return append([]string(nil), hp.all...)
}

// AddBackend adds a new backend to the pool.
func (hp *HealthyPool) AddBackend(backend string) {
	hp.mu.Lock()
	defer hp.mu.Unlock()
	hp.all = append(hp.all, backend)
	hp.checker.active.AddBackend(backend)
}

// RemoveBackend removes a backend from the pool.
func (hp *HealthyPool) RemoveBackend(backend string) {
	hp.mu.Lock()
	defer hp.mu.Unlock()

	for i, b := range hp.all {
		if b == backend {
			hp.all = append(hp.all[:i], hp.all[i+1:]...)
			break
		}
	}
	hp.checker.active.RemoveBackend(backend)
}