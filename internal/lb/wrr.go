package lb

import "sync"

// WeightedBackend pairs a backend address with its weight.
type WeightedBackend struct {
	Addr   string
	Weight int
}

// weightedEntry tracks the dynamic current weight for each backend.
type weightedEntry struct {
	addr          string
	weight        int // fixed, configured weight
	currentWeight int // changes every round
}

// WeightedRoundRobin implements smooth weighted round robin (nginx algorithm).
//
// Each call to Next():
//  1. Add each backend's fixed weight to its current weight
//  2. Pick the backend with the highest current weight
//  3. Subtract total weight from the picked backend's current weight
//
// This spreads requests smoothly rather than bursting.
type WeightedRoundRobin struct {
	mu          sync.Mutex
	entries     []weightedEntry
	totalWeight int
}

// NewWeightedRoundRobin creates a new smooth weighted round robin balancer.
// Backends with Weight <= 0 default to 1.
func NewWeightedRoundRobin(backends []WeightedBackend) *WeightedRoundRobin {
	entries := make([]weightedEntry, len(backends))
	total := 0

	for i, b := range backends {
		w := b.Weight
		if w <= 0 {
			w = 1
		}
		entries[i] = weightedEntry{
			addr:          b.Addr,
			weight:        w,
			currentWeight: 0,
		}
		total += w
	}

	return &WeightedRoundRobin{
		entries:     entries,
		totalWeight: total,
	}
}

// Next returns the next backend address using smooth weighted round robin.
func (wrr *WeightedRoundRobin) Next() string {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()

	if len(wrr.entries) == 0 {
		return ""
	}

	bestIdx := 0

	// Step 1 & 2: add fixed weight, track highest
	for i := range wrr.entries {
		wrr.entries[i].currentWeight += wrr.entries[i].weight

		if wrr.entries[i].currentWeight > wrr.entries[bestIdx].currentWeight {
			bestIdx = i
		}
	}

	// Step 3: subtract total from the picked one
	wrr.entries[bestIdx].currentWeight -= wrr.totalWeight

	return wrr.entries[bestIdx].addr
}