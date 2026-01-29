# Milestone 3.2: Per-Client Limiting

**Phase:** 3 — Rate Limiting
**Status:** [ ] Not started

## Goal

Maintain a separate token bucket per client so one abusive client doesn't exhaust the rate limit for everyone.

## Key Concepts

- **Client identification** — Key by client IP, API key, or user ID.
- **Bucket map** — `map[string]*TokenBucket` with concurrent access.
- **Garbage collection** — Stale buckets (clients that stopped sending) must be cleaned up to prevent memory leaks.
- **RWMutex** — Read-heavy workload (most requests hit existing buckets), so `sync.RWMutex` is better than `sync.Mutex`.

## Requirements

- [ ] Map of client key to token bucket
- [ ] Create bucket on first request from a client
- [ ] Thread-safe map access (RWMutex)
- [ ] Background goroutine to garbage collect stale buckets
- [ ] Configurable stale threshold (e.g., remove after 10 minutes of inactivity)
- [ ] Configurable client key extraction (IP, header, etc.)

## Questions to Answer Before Coding

1. Why `sync.RWMutex` instead of `sync.Mutex` for the bucket map?
2. How do you safely garbage collect while other goroutines are accessing the map?
3. What happens under memory pressure with millions of unique client IPs?
4. Why not use `sync.Map` here?
