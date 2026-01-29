# Milestone 7.1: Metrics (Prometheus)

**Phase:** 7 — Observability
**Status:** [ ] Not started

## Goal

Expose gateway metrics in Prometheus format so you can monitor request rates, latencies, backend health, and rate limiting.

## Key Concepts

- **Counters** — Monotonically increasing values (total requests, total errors).
- **Histograms** — Distribution of values in buckets (latency percentiles).
- **Gauges** — Point-in-time values (active connections, circuit state).
- **Labels** — Dimensions for slicing metrics (service, backend, status code).
- **`/metrics` endpoint** — Standard Prometheus scrape endpoint.

## Requirements

- [ ] `gateway_requests_total` counter with labels: service, status, method
- [ ] `gateway_request_duration_seconds` histogram with service label
- [ ] `gateway_backend_healthy` gauge per backend
- [ ] `gateway_rate_limited_total` counter per client
- [ ] `gateway_circuit_state` gauge per backend (0=closed, 1=open, 2=half-open)
- [ ] `gateway_active_connections` gauge per backend
- [ ] Expose `/metrics` HTTP endpoint

## Questions to Answer Before Coding

1. Why use a histogram instead of a gauge for latency?
2. What histogram bucket boundaries make sense for an API gateway?
3. What's the cardinality concern with labels like `client_ip`?
4. How does Prometheus scraping work?
