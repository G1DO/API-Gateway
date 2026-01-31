package health

// CombinedChecker combines active and passive health checks.
//
// A backend is considered healthy only if BOTH active and passive checks pass.
// This provides defense-in-depth: active catches idle failures, passive
// catches under-load failures.
type CombinedChecker struct {
	active  *ActiveChecker
	passive *PassiveChecker
}

// NewCombined creates a combined health checker with both active and passive checks.
func NewCombined(active *ActiveChecker, passive *PassiveChecker) *CombinedChecker {
	return &CombinedChecker{
		active:  active,
		passive: passive,
	}
}

// IsHealthy returns true only if both active and passive checks pass.
func (c *CombinedChecker) IsHealthy(backend string) bool {
	return c.active.IsHealthy(backend) && c.passive.IsHealthy(backend)
}

// RecordSuccess records a successful request (for passive checks).
func (c *CombinedChecker) RecordSuccess(backend string) {
	c.passive.RecordSuccess(backend)
}

// RecordFailure records a failed request (for passive checks).
func (c *CombinedChecker) RecordFailure(backend string) {
	c.passive.RecordFailure(backend)
}

// ActiveStatus returns the active health check status.
func (c *CombinedChecker) ActiveStatus(backend string) Status {
	return c.active.Status(backend)
}

// PassiveErrorRate returns the passive error rate.
func (c *CombinedChecker) PassiveErrorRate(backend string) float64 {
	return c.passive.ErrorRate(backend)
}

// Close stops the active health checker.
func (c *CombinedChecker) Close() {
	c.active.Close()
}