# Milestone 5.1: Active Health Checks

**Phase:** 5 — Health Checking
**Status:** [x] Complete

## Goal

Periodically probe each backend with a health endpoint to detect failures before real traffic hits them.

## Key Concepts

- **Active probing** — Gateway sends periodic GET requests to a health endpoint (e.g., `/health`).
- **Consecutive thresholds** — Mark unhealthy after N consecutive failures. Mark healthy after M consecutive successes. Prevents flapping.
- **Probe interval** — Too frequent = unnecessary load. Too infrequent = slow detection.
- **Timeout per probe** — Health check must have its own timeout, shorter than request timeout.

## Requirements

- [ ] Background goroutine that probes backends at a configurable interval
- [ ] Configurable health endpoint path per backend
- [ ] Configurable probe timeout
- [ ] Mark unhealthy after N consecutive failures
- [ ] Mark healthy after M consecutive successes
- [ ] Use `context.Context` for cancellation (graceful shutdown)
- [ ] Thread-safe health status updates

## Questions to Answer Before Coding

1. Why use consecutive failure/success counts instead of a single check?
2. What constitutes a "healthy" response? Just 200? Any 2xx?
3. How does the health checker communicate status to the load balancer?
4. What happens during gateway startup before the first health check completes?
5. Why should health check timeout be shorter than request timeout?
