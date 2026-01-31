package ratelimit

import (
	"sync"
	"time"
)

// clientEntry holds a token bucket and the last time it was accessed.
type clientEntry struct {
	bucket     *TokenBucket
	lastAccess time.Time
}

// PerClient maintains a separate token bucket per client key (IP, API key, etc.).
//
// A background goroutine garbage-collects buckets that have been idle
// longer than staleThreshold to prevent unbounded memory growth.
type PerClient struct {
	mu             sync.RWMutex
	clients        map[string]*clientEntry
	capacity       int
	rate           float64
	staleThreshold time.Duration
	stop           chan struct{}
}

// KeyFunc extracts a client identifier from an HTTP request.
// Common implementations: extract client IP, API key header, etc.
type KeyFunc func(r interface{}) string

// NewPerClient creates a per-client rate limiter. Each new client gets a
// token bucket with the given capacity and rate. Buckets idle longer than
// staleThreshold are garbage collected.
func NewPerClient(capacity int, rate float64, staleThreshold time.Duration) *PerClient {
	pc := &PerClient{
		clients:        make(map[string]*clientEntry),
		capacity:       capacity,
		rate:           rate,
		staleThreshold: staleThreshold,
		stop:           make(chan struct{}),
	}
	go pc.gc()
	return pc
}

// Allow checks the rate limit for the given client key.
// Creates a new bucket on first request from a client.
func (pc *PerClient) Allow(key string) (ok bool, retryAfter time.Duration) {
	// Fast path: bucket already exists
	pc.mu.RLock()
	entry, exists := pc.clients[key]
	pc.mu.RUnlock()

	if exists {
		entry.lastAccess = time.Now()
		return entry.bucket.Allow()
	}

	// Slow path: create new bucket
	pc.mu.Lock()
	// Double-check after acquiring write lock
	entry, exists = pc.clients[key]
	if exists {
		pc.mu.Unlock()
		entry.lastAccess = time.Now()
		return entry.bucket.Allow()
	}

	entry = &clientEntry{
		bucket:     NewTokenBucket(pc.capacity, pc.rate),
		lastAccess: time.Now(),
	}
	pc.clients[key] = entry
	pc.mu.Unlock()

	return entry.bucket.Allow()
}

// gc periodically removes stale client buckets.
func (pc *PerClient) gc() {
	ticker := time.NewTicker(pc.staleThreshold / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pc.mu.Lock()
			now := time.Now()
			for key, entry := range pc.clients {
				if now.Sub(entry.lastAccess) > pc.staleThreshold {
					delete(pc.clients, key)
				}
			}
			pc.mu.Unlock()
		case <-pc.stop:
			return
		}
	}
}

// Close stops the background garbage collection goroutine.
func (pc *PerClient) Close() {
	close(pc.stop)
}