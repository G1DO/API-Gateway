package lb

import "sync/atomic"

// leastConnEntry tracks active connections for a single backend.
type leastConnEntry struct {
	addr   string
	active atomic.Int64
}

// LeastConnections picks the backend with the fewest active connections.
//
// Usage:
//
//	addr := lc.Next()       // picks backend, increments its counter
//	defer lc.Done(addr)     // decrements counter when request is done
//
// The caller MUST call Done() when the request completes (success or error),
// otherwise the counter leaks and the backend appears permanently busy.
type LeastConnections struct {
	entries []leastConnEntry
}

// NewLeastConnections creates a new least-connections balancer.
func NewLeastConnections(backends []string) *LeastConnections {
	entries := make([]leastConnEntry, len(backends))
	for i, addr := range backends {
		entries[i].addr = addr
	}
	return &LeastConnections{entries: entries}
}

// Next returns the backend with the fewest active connections
// and increments its active count.
func (lc *LeastConnections) Next() string {
	if len(lc.entries) == 0 {
		return ""
	}

	bestIdx := 0
	bestCount := lc.entries[0].active.Load()

	for i := 1; i < len(lc.entries); i++ {
		count := lc.entries[i].active.Load()
		if count < bestCount {
			bestCount = count
			bestIdx = i
		}
	}

	lc.entries[bestIdx].active.Add(1)
	return lc.entries[bestIdx].addr
}

// Done decrements the active connection count for the given backend.
// Must be called when a request completes (success or error).
func (lc *LeastConnections) Done(addr string) {
	for i := range lc.entries {
		if lc.entries[i].addr == addr {
			lc.entries[i].active.Add(-1)
			return
		}
	}
}
