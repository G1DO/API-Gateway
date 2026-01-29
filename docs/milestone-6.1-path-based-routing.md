# Milestone 6.1: Path-Based Routing

**Phase:** 6 — Routing & Configuration
**Status:** [ ] Not started

## Goal

Route incoming requests to different backend services based on the URL path.

## Key Concepts

- **Path matching** — Exact match, prefix match, wildcard match (e.g., `/api/users/*`).
- **Route priority** — More specific routes should match before general ones.
- **Service abstraction** — A route maps to a "service" which has its own set of backends and load balancer.
- **YAML configuration** — Routes defined in a config file, not hardcoded.

## Requirements

- [ ] Parse route rules from YAML configuration
- [ ] Match incoming request path to a route
- [ ] Support prefix matching with wildcards (`/api/users/*`)
- [ ] Route priority: longer/more specific prefixes match first
- [ ] Each route points to a service with its own backend pool
- [ ] Return 404 for unmatched paths

## Questions to Answer Before Coding

1. How do you efficiently match a path against many route rules?
2. What's the correct priority order when multiple routes could match?
3. Should path matching be case-sensitive?
4. How does the router integrate with the rest of the middleware chain?
