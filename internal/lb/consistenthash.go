package lb

import (
	"fmt"
	"hash/crc32"
	"sort"
	"sync"
)

// ConsistentHash maps request keys to backends using a hash ring.
//
// How it works:
//   - Each backend gets multiple "virtual nodes" placed on a hash ring (0 to 2^32-1)
//   - To route a request: hash the key, walk clockwise to the nearest virtual node
//   - That virtual node maps back to a real backend
//
// When a backend is added/removed, only ~1/N of keys remap (N = number of backends).
type ConsistentHash struct {
	mu       sync.RWMutex
	ring     []uint32            // sorted hash values (virtual nodes)
	nodeMap  map[uint32]string   // hash value -> backend address
	replicas int                 // virtual nodes per backend
}

// NewConsistentHash creates a hash ring with the given number of virtual nodes
// per backend. 150 is a reasonable default for good distribution.
func NewConsistentHash(replicas int, backends []string) *ConsistentHash {
	ch := &ConsistentHash{
		replicas: replicas,
		nodeMap:  make(map[uint32]string),
	}
	for _, b := range backends {
		ch.add(b)
	}
	sort.Slice(ch.ring, func(i, j int) bool { return ch.ring[i] < ch.ring[j] })
	return ch
}

// add places virtual nodes for a backend on the ring (not thread-safe, used during construction).
func (ch *ConsistentHash) add(addr string) {
	for i := 0; i < ch.replicas; i++ {
		key := fmt.Sprintf("%s-%d", addr, i)
		h := crc32.ChecksumIEEE([]byte(key))
		ch.ring = append(ch.ring, h)
		ch.nodeMap[h] = addr
	}
}

// Next hashes the given key and returns the closest backend clockwise on the ring.
// The key is typically a client IP or request attribute.
//
// Note: this takes a key parameter but the Balancer interface takes no args.
// Use NextWithKey for consistent hashing; Next() is provided for interface
// compatibility but uses an empty key (always returns the same backend).
func (ch *ConsistentHash) Next() string {
	return ch.NextWithKey("")
}

// NextWithKey returns the backend for a specific key.
func (ch *ConsistentHash) NextWithKey(key string) string {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	if len(ch.ring) == 0 {
		return ""
	}

	h := crc32.ChecksumIEEE([]byte(key))

	// Binary search for the first virtual node with hash >= h
	idx := sort.Search(len(ch.ring), func(i int) bool {
		return ch.ring[i] >= h
	})

	// Wrap around: if past the end, go to the first node (it's a ring)
	if idx == len(ch.ring) {
		idx = 0
	}

	return ch.nodeMap[ch.ring[idx]]
}