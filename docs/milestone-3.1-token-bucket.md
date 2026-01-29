# Milestone 3.1: Token Bucket Algorithm

**Phase:** 3 — Rate Limiting
**Status:** [ ] Not started

## Goal

Implement the token bucket algorithm to cap the rate of incoming requests.

## Key Concepts

- **Token bucket** — A bucket holds tokens up to a max capacity. Tokens refill at a fixed rate. Each request consumes one token. No tokens = reject with 429.
- **Burst vs sustained rate** — Capacity = max burst size. Refill rate = sustained throughput.
- **Lazy refill** — Don't use a ticker. Calculate tokens based on elapsed time when a request arrives.
- **Mutex for state** — Tokens and last-refill time are shared state.

## Requirements

- [ ] Token bucket with configurable capacity and refill rate
- [ ] `Allow()` method returns true/false
- [ ] Lazy token refill (calculate on each call, not with a background ticker)
- [ ] Thread-safe with mutex
- [ ] Return 429 Too Many Requests when tokens exhausted
- [ ] Include `Retry-After` header in 429 responses

## Questions to Answer Before Coding

1. Why lazy refill instead of a background goroutine adding tokens?
2. What's the relationship between capacity (burst) and rate (sustained)?
3. Why use `float64` for tokens instead of `int`?
4. What should the `Retry-After` header value be?
