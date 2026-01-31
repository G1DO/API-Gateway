# Milestone 6.2: Header-Based Routing

**Phase:** 6 — Routing & Configuration
**Status:** [x] Complete

## Goal

Route requests based on HTTP headers, enabling virtual hosts, API versioning, and canary deployments.

## Key Concepts

- **Host header** — Virtual hosting: different domains routed to different services on the same gateway.
- **Custom headers** — Route by `X-API-Version`, `X-Canary`, etc.
- **A/B testing** — Route a percentage of traffic to a canary backend based on headers.
- **Combined matching** — Match on both path AND headers.

## Requirements

- [x] Match routes by `Host` header (virtual hosts)
- [x] Match routes by custom headers
- [x] Combined path + header matching
- [x] Header matching supports exact match and presence check (`"*"` = presence only)
- [x] YAML configuration for header-based routes

## Implementation

- **File:** `internal/router/router.go` -- `matchHeaders()` checks all required headers
- Header value `"*"` means presence check (any non-empty value matches)
- Otherwise exact string match via `req.Header.Get(key)`
- Routes with headers sort before routes without at the same path length (more specific first)
- All specified headers must match (AND logic)

## Questions to Answer Before Coding

1. How do you handle a request that matches path but not headers?
2. What order should matching happen: path first or headers first?
3. How would you implement percentage-based canary routing?
4. How does header-based routing interact with path-based routing priority?
