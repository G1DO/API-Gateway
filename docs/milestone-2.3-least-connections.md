# Milestone 2.3: Least Connections

**Phase:** 2 — Load Balancing
**Status:** [ ] Not started

## Goal

Route each request to the backend with the fewest active connections, adapting to real-time load.

## Key Concepts

- **Active connection tracking** — Increment when request starts, decrement when it completes.
- **Atomics for counters** — `atomic.Int64` for lock-free increment/decrement.
- **Tie-breaking** — When multiple backends have the same count, pick one (round robin among ties, or random).

## Requirements

- [ ] Track active connections per backend using atomic counters
- [ ] Pick the backend with the lowest active connection count
- [ ] Increment on request start, decrement on request completion (even on error)
- [ ] Handle tie-breaking deterministically
- [ ] Implements the `Balancer` interface

## Questions to Answer Before Coding

1. Why is least connections better than round robin for variable-latency backends?
2. What happens if you forget to decrement the counter on error paths?
3. Why use `atomic.Int64` instead of a mutex here?
4. What's the race condition risk when reading all counters to find the minimum?
