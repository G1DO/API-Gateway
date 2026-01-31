# Milestone 1.1: Basic Proxy

**Phase:** 1 — Reverse Proxy
**Status:** [x] Complete

## Goal

Build the simplest possible HTTP proxy: accept a request, forward it to a single hardcoded backend, return the response to the client.

## Key Concepts

- **Hop-by-hop headers** — Connection, Keep-Alive, Transfer-Encoding, Proxy-Authenticate, Proxy-Authorization, TE, Trailer, Upgrade. These must NOT be forwarded to the backend.
- **Request/response streaming** — Don't buffer the entire body in memory. Stream it through.
- **Context propagation** — Use Go's `context` to propagate timeouts and cancellation from client to backend request.
- **Connection: keep-alive vs close** — Understand what each means and how the proxy should handle them.

## Requirements

- [ ] Accept incoming HTTP requests on a configurable port
- [ ] Create an outbound request to a hardcoded backend URL
- [ ] Copy request headers from client to backend (excluding hop-by-hop headers)
- [ ] Forward request body via streaming (not buffering)
- [ ] Copy response status code, headers, and body back to client
- [ ] Handle Connection header correctly (keep-alive vs close)
- [ ] Properly handle errors (backend unreachable, timeouts)

## Questions to Answer Before Coding

1. What are hop-by-hop headers and why must a proxy strip them?
2. What happens if you forward the `Connection: keep-alive` header to the backend?
3. Why is streaming the body important instead of reading it all into memory?
4. What does `http.Request.Clone()` do vs manually constructing a new request?
5. How does Go's `context.Context` help with timeout propagation?
