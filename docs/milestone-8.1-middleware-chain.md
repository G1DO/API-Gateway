# Milestone 8.1: Middleware Chain Architecture

**Phase:** 8 — Production Readiness
**Status:** [ ] Not started

## Goal

Refactor the gateway to use a composable middleware pattern where rate limiting, circuit breaking, tracing, logging, and metrics are independent functions chained together — not hardcoded in a monolithic handler. This is how real proxies (Envoy, Kong, Caddy) work.

## Key Concepts

- **Middleware signature** — `func(http.Handler) http.Handler`. Each middleware wraps the next handler, forming a chain.
- **Chain composition** — `Chain(tracing, logging, rateLimit, circuitBreaker, proxy)` builds the full pipeline. Order matters.
- **Context propagation** — Middleware passes data to downstream handlers via `context.Context` (trace ID, client IP, start time).
- **Separation of concerns** — Each middleware does one thing. Easy to add, remove, or reorder without touching other code.

## Requirements

- [ ] Define `Middleware` type as `func(http.Handler) http.Handler`
- [ ] Implement `Chain(middlewares ...Middleware) Middleware` that composes them in order
- [ ] Wrap rate limiter as middleware (reject with 429 + Retry-After header)
- [ ] Wrap circuit breaker as middleware (reject with 503 when circuit is open)
- [ ] Wrap tracing as middleware (generate/propagate X-Request-ID)
- [ ] Wrap structured logging as middleware (log method, path, status, latency per request)
- [ ] Wrap metrics as middleware (increment counters, observe latency histogram)
- [ ] Wire the chain in `main.go`: `Chain(tracing, logging, metrics, rateLimit, circuitBreaker)(proxy)`

## Questions to Answer Before Coding

1. Why does middleware ordering matter? What breaks if you put logging before tracing vs after?
2. How do you pass data between middleware (e.g., trace ID generated in tracing middleware, used in logging middleware)?
3. How does the `func(http.Handler) http.Handler` pattern enable composition without each middleware knowing about the others?
4. How do real proxies like Envoy and Kong implement their filter/plugin chains?
5. How do you capture the response status code in a logging middleware when `http.ResponseWriter` doesn't expose it?
