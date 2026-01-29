# Milestone 1.2: Connection Pooling

**Phase:** 1 — Reverse Proxy
**Status:** [ ] Not started

## Goal

Reuse TCP connections to backends instead of opening a new connection per request. This eliminates TCP and TLS handshake overhead.

## Key Concepts

- **TCP handshake cost** — Each new connection costs 1 RTT. With TLS, 2+ RTT. Pooling avoids this on subsequent requests.
- **Idle connections** — Pooled connections sitting unused. Must be closed after a timeout to free resources.
- **Pool size limits** — Too few connections = bottleneck. Too many = resource waste on backend.
- **Go's `http.Transport`** — Understand `MaxIdleConns`, `MaxIdleConnsPerHost`, `IdleConnTimeout`.

## Requirements

- [ ] Reuse TCP connections to the same backend across requests
- [ ] Configurable max pool size per backend
- [ ] Idle timeout — close connections unused for too long
- [ ] Properly drain response bodies to allow connection reuse
- [ ] Verify pooling works (log connection reuse vs new connections)

## Questions to Answer Before Coding

1. What happens if you don't fully read and close the response body?
2. Why does Go's `http.Transport` require you to drain the body for connection reuse?
3. What's the difference between `MaxIdleConns` and `MaxIdleConnsPerHost`?
4. How do you decide the right pool size for a backend?
5. What happens when all pooled connections are in use and a new request arrives?
