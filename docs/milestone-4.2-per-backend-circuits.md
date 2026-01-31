# Milestone 4.2: Per-Backend Circuits

**Phase:** 4 — Circuit Breaker
**Status:** [x] Complete

## Goal

Assign a separate circuit breaker to each backend so one failing backend doesn't cause the gateway to reject requests to healthy backends.

## Key Concepts

- **Isolation** — Backend A failing should not affect traffic to Backend B.
- **Integration with load balancer** — When a circuit is open, the load balancer should skip that backend.
- **Recovery** — When a circuit closes (backend recovers), gradually reintroduce it to avoid overload.

## Requirements

- [ ] Map of backend address to circuit breaker
- [ ] Load balancer skips backends with open circuits
- [ ] Create circuit breaker lazily on first request to a backend
- [ ] Circuit state visible for monitoring/metrics
- [ ] Coordinate with health checker (Phase 5) — if health check passes, consider closing circuit

## Questions to Answer Before Coding

1. How should the load balancer behave when ALL backends have open circuits?
2. Should circuit breakers be created eagerly (at config time) or lazily (on first request)?
3. How do circuit breakers and health checks complement each other?
4. What happens to in-flight requests when a circuit transitions to Open?
