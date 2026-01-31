package lb

import (
	"fmt"
	"math"
	"sync"
	"testing"
)

// --- Round Robin ---

func TestRoundRobinCycles(t *testing.T) {
	backends := []string{"A", "B", "C"}
	rr := NewRoundRobin(backends)

	// Should cycle A, B, C, A, B, C...
	// Note: atomic.AddUint64 starts at 1, so first pick is index 1
	for i := 0; i < 9; i++ {
		got := rr.Next()
		want := backends[(i+1)%3]
		if got != want {
			t.Errorf("call %d: got %s, want %s", i, got, want)
		}
	}
}

func TestRoundRobinDistribution(t *testing.T) {
	backends := []string{"A", "B", "C"}
	rr := NewRoundRobin(backends)
	counts := map[string]int{}

	for i := 0; i < 300; i++ {
		counts[rr.Next()]++
	}

	for _, b := range backends {
		if counts[b] != 100 {
			t.Errorf("expected 100 for %s, got %d", b, counts[b])
		}
	}
}

func TestRoundRobinConcurrent(t *testing.T) {
	backends := []string{"A", "B", "C"}
	rr := NewRoundRobin(backends)

	var wg sync.WaitGroup
	mu := sync.Mutex{}
	counts := map[string]int{}

	for i := 0; i < 300; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			addr := rr.Next()
			mu.Lock()
			counts[addr]++
			mu.Unlock()
		}()
	}
	wg.Wait()

	for _, b := range backends {
		if counts[b] != 100 {
			t.Errorf("expected 100 for %s, got %d", b, counts[b])
		}
	}
}

// --- Weighted Round Robin ---

func TestWRRDistribution(t *testing.T) {
	backends := []WeightedBackend{
		{Addr: "A", Weight: 5},
		{Addr: "B", Weight: 1},
		{Addr: "C", Weight: 1},
	}
	wrr := NewWeightedRoundRobin(backends)
	counts := map[string]int{}

	total := 700
	for i := 0; i < total; i++ {
		counts[wrr.Next()]++
	}

	// A should get 5/7, B and C should get 1/7 each
	if counts["A"] != 500 {
		t.Errorf("A: expected 500, got %d", counts["A"])
	}
	if counts["B"] != 100 {
		t.Errorf("B: expected 100, got %d", counts["B"])
	}
	if counts["C"] != 100 {
		t.Errorf("C: expected 100, got %d", counts["C"])
	}
}

func TestWRRSmooth(t *testing.T) {
	// With weights A=2, B=1, the sequence should be A,B,A,A,B,A,...
	// NOT A,A,B,A,A,B — it should spread A out.
	backends := []WeightedBackend{
		{Addr: "A", Weight: 2},
		{Addr: "B", Weight: 1},
	}
	wrr := NewWeightedRoundRobin(backends)

	// First 3 picks should be: A, B, A (smooth)
	// NOT: A, A, B (burst)
	results := make([]string, 3)
	for i := 0; i < 3; i++ {
		results[i] = wrr.Next()
	}

	if results[0] != "A" || results[1] != "B" || results[2] != "A" {
		t.Errorf("expected [A B A], got %v", results)
	}
}

func TestWRRDefaultWeight(t *testing.T) {
	backends := []WeightedBackend{
		{Addr: "A", Weight: 0},  // should default to 1
		{Addr: "B", Weight: -1}, // should default to 1
	}
	wrr := NewWeightedRoundRobin(backends)
	counts := map[string]int{}

	for i := 0; i < 100; i++ {
		counts[wrr.Next()]++
	}

	if counts["A"] != 50 || counts["B"] != 50 {
		t.Errorf("expected 50/50, got A=%d B=%d", counts["A"], counts["B"])
	}
}

func TestWRRConcurrent(t *testing.T) {
	backends := []WeightedBackend{
		{Addr: "A", Weight: 3},
		{Addr: "B", Weight: 1},
	}
	wrr := NewWeightedRoundRobin(backends)

	var wg sync.WaitGroup
	mu := sync.Mutex{}
	counts := map[string]int{}
	total := 400

	for i := 0; i < total; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			addr := wrr.Next()
			mu.Lock()
			counts[addr]++
			mu.Unlock()
		}()
	}
	wg.Wait()

	if counts["A"] != 300 {
		t.Errorf("A: expected 300, got %d", counts["A"])
	}
	if counts["B"] != 100 {
		t.Errorf("B: expected 100, got %d", counts["B"])
	}
}

// --- Least Connections ---

func TestLeastConnPicksLowest(t *testing.T) {
	lc := NewLeastConnections([]string{"A", "B", "C"})

	// All at 0 — should pick first (A)
	got := lc.Next()
	if got != "A" {
		t.Fatalf("expected A, got %s", got)
	}
	// A now has 1 active, B and C have 0 — should pick B
	got = lc.Next()
	if got != "B" {
		t.Fatalf("expected B, got %s", got)
	}
	// A=1, B=1, C=0 — should pick C
	got = lc.Next()
	if got != "C" {
		t.Fatalf("expected C, got %s", got)
	}
}

func TestLeastConnDoneDecrement(t *testing.T) {
	lc := NewLeastConnections([]string{"A", "B"})

	// Pick A twice
	lc.Next() // A=1
	lc.Next() // B=1

	// Release A
	lc.Done("A") // A=0, B=1

	// Should pick A again since it has fewer connections
	got := lc.Next()
	if got != "A" {
		t.Fatalf("expected A after Done, got %s", got)
	}
}

func TestLeastConnConcurrent(t *testing.T) {
	lc := NewLeastConnections([]string{"A", "B", "C"})
	var wg sync.WaitGroup

	for i := 0; i < 300; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			addr := lc.Next()
			// Simulate work
			lc.Done(addr)
		}()
	}
	wg.Wait()

	// After all done, all counts should be 0
	for i := range lc.entries {
		count := lc.entries[i].active.Load()
		if count != 0 {
			t.Errorf("%s: expected 0 active, got %d", lc.entries[i].addr, count)
		}
	}
}

// --- Consistent Hash ---

func TestConsistentHashSameKeysSameBackend(t *testing.T) {
	ch := NewConsistentHash(150, []string{"A", "B", "C"})

	// Same key should always return same backend
	first := ch.NextWithKey("user-123")
	for i := 0; i < 100; i++ {
		got := ch.NextWithKey("user-123")
		if got != first {
			t.Fatalf("key 'user-123' mapped to %s then %s", first, got)
		}
	}
}

func TestConsistentHashDistribution(t *testing.T) {
	ch := NewConsistentHash(150, []string{"A", "B", "C"})
	counts := map[string]int{}

	for i := 0; i < 3000; i++ {
		key := fmt.Sprintf("key-%d", i)
		counts[ch.NextWithKey(key)]++
	}

	// With 150 virtual nodes and 3 backends, each should get roughly 1/3
	// Allow 20% deviation
	expected := 1000.0
	for _, b := range []string{"A", "B", "C"} {
		deviation := math.Abs(float64(counts[b])-expected) / expected
		if deviation > 0.25 {
			t.Errorf("%s: got %d (%.0f%% deviation from expected %d)",
				b, counts[b], deviation*100, int(expected))
		}
	}
}

func TestConsistentHashMinimalRemapping(t *testing.T) {
	backends3 := []string{"A", "B", "C"}
	backends4 := []string{"A", "B", "C", "D"}

	ch3 := NewConsistentHash(150, backends3)
	ch4 := NewConsistentHash(150, backends4)

	remapped := 0
	total := 1000
	for i := 0; i < total; i++ {
		key := fmt.Sprintf("key-%d", i)
		if ch3.NextWithKey(key) != ch4.NextWithKey(key) {
			remapped++
		}
	}

	// Adding 1 backend to 3 should remap roughly 1/4 of keys
	// Allow some margin — anything under 50% is acceptable
	ratio := float64(remapped) / float64(total)
	if ratio > 0.50 {
		t.Errorf("%.0f%% of keys remapped when adding a backend (expected ~25%%)", ratio*100)
	}
}

func TestConsistentHashEmptyReturnsEmpty(t *testing.T) {
	ch := NewConsistentHash(150, nil)
	if got := ch.NextWithKey("anything"); got != "" {
		t.Fatalf("expected empty string, got %s", got)
	}
}