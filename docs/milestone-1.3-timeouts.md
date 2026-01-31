# Milestone 1.3: Timeouts

**Phase:** 1 — Reverse Proxy
**Status:** [x] Complete

## Goal

Add configurable timeouts to prevent the proxy from hanging indefinitely when backends are slow or unreachable.

## Key Concepts

- **Connection timeout** — Max time to establish a TCP connection to the backend.
- **Request timeout** — Max total time for the entire request/response cycle.
- **Idle timeout** — Max time a pooled connection can sit unused before being closed.
- **Backpressure** — When backends are slower than clients, requests queue up. Timeouts prevent unbounded queuing.

## Requirements

- [ ] Configurable connection timeout (e.g., 5s)
- [ ] Configurable request timeout (e.g., 30s)
- [ ] Configurable idle timeout for pooled connections (e.g., 90s)
- [ ] Return appropriate error (504 Gateway Timeout) when backend times out
- [ ] Use `context.WithTimeout` for request-scoped timeouts
- [ ] Timeout config struct that can be loaded from configuration

## Questions to Answer Before Coding

1. What's the difference between connection timeout and request timeout?
2. Why is 504 the right status code for a backend timeout (not 408)?
3. What happens to the backend request if the client disconnects mid-request?
4. How does `context.WithTimeout` interact with `http.Client.Timeout`?
5. What's a reasonable default for each timeout type and why?
