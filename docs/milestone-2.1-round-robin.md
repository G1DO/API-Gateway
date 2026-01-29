# Milestone 2.1: Round Robin

**Phase:** 2 — Load Balancing
**Status:** [ ] Not started

## Goal

Distribute requests evenly across multiple backends by rotating through them sequentially.

## Key Concepts

- **Round robin** — Simplest load balancing: backend A, B, C, A, B, C...
- **Thread safety** — Multiple goroutines picking backends concurrently. The counter must be safe.
- **Atomic operations** — Using `atomic.AddUint64` vs mutex for the counter.

## Requirements

- [ ] Maintain an ordered list of backends
- [ ] Rotate through backends sequentially per request
- [ ] Thread-safe counter (concurrent requests must not corrupt state)
- [ ] Wrap around when reaching the end of the list
- [ ] Implement a `Balancer` interface that other strategies will also use

## Questions to Answer Before Coding

1. Why is round robin insufficient for backends with different capacities?
2. What happens if a backend in the rotation is unhealthy?
3. Why use `atomic.AddUint64` instead of a mutex-protected counter?
4. What interface should all load balancing strategies share?
