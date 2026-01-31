# Milestone 6.1: Path-Based Routing

**Phase:** 6 — Routing & Configuration
**Status:** [x] Complete

## Goal

Route incoming requests to different backend services based on the URL path.

## Key Concepts

- **Path matching** — Exact match, prefix match, wildcard match (e.g., `/api/users/*`).
- **Route priority** — More specific routes should match before general ones.
- **Service abstraction** — A route maps to a "service" which has its own set of backends and load balancer.
- **YAML configuration** — Routes defined in a config file, not hardcoded.

## Requirements

- [x] Parse route rules from YAML configuration
- [x] Match incoming request path to a route
- [x] Support prefix matching with wildcards (`/api/users/*`)
- [x] Route priority: longer/more specific prefixes match first
- [x] Each route points to a service with its own backend pool
- [x] Return nil for unmatched paths (caller decides response)

## Implementation

- **File:** `internal/router/config.go` -- YAML parsing via `gopkg.in/yaml.v3`, validation (non-empty path, at least one backend)
- **File:** `internal/router/router.go` -- Prefix matching with routes sorted by specificity (longest path first)
- Trailing `/*` and `*` stripped from config paths for clean prefix matching
- `Router.Match()` returns `*Route` or nil

## Questions to Answer Before Coding

1. How do you efficiently match a path against many route rules?
2. What's the correct priority order when multiple routes could match?
3. Should path matching be case-sensitive?
4. How does the router integrate with the rest of the middleware chain?
