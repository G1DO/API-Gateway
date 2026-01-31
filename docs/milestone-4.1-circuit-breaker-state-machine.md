# Milestone 4.1: Circuit Breaker State Machine

**Phase:** 4 — Circuit Breaker
**Status:** [x] Complete

## Goal

Implement a circuit breaker that stops calling failing backends, preventing cascade failures. Uses a three-state machine: Closed, Open, Half-Open.

## Key Concepts

- **Cascade failure** — Backend B is slow, proxy threads pile up waiting, proxy runs out of resources, everything dies.
- **Fail fast** — When a backend is known to be down, reject immediately instead of waiting for a timeout.
- **Three states:**
  - **Closed** — Normal. Requests pass through. Failures are counted.
  - **Open** — Tripped. All requests rejected immediately. No backend calls.
  - **Half-Open** — After a timeout, allow ONE test request. Success = close circuit. Failure = reopen.
- **Thundering herd** — When circuit closes, don't flood the backend with all queued requests at once.

## Requirements

- [ ] Three-state machine: Closed, Open, Half-Open
- [ ] Configurable failure threshold (e.g., 5 consecutive failures)
- [ ] Configurable timeout for Open → Half-Open transition
- [ ] `Allow()` — check if request should proceed
- [ ] `RecordSuccess()` — reset failure count, transition to Closed
- [ ] `RecordFailure()` — increment failures, potentially transition to Open
- [ ] Thread-safe state transitions
- [ ] In Half-Open, allow only ONE request through (not all waiting requests)

## Questions to Answer Before Coding

1. Why is fail-fast better than letting every request timeout against a dead backend?
2. What counts as a "failure"? Timeout? 5xx? Connection refused? All of these?
3. Why allow only one request in Half-Open instead of all pending requests?
4. What's the thundering herd problem and how does Half-Open mitigate it?
5. Should the failure counter use consecutive failures or a failure rate over a window?
