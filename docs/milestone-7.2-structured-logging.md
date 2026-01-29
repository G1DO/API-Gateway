# Milestone 7.2: Structured Logging

**Phase:** 7 — Observability
**Status:** [ ] Not started

## Goal

Replace unstructured text logs with JSON-formatted structured logs for machine parseability and searchability.

## Key Concepts

- **Structured vs unstructured** — `{"level":"info","path":"/api/users","status":200}` vs `INFO: GET /api/users 200`.
- **Log levels** — Debug, Info, Warn, Error. Configurable minimum level.
- **Request context** — Every log entry for a request should include: method, path, service, backend, status, latency, client IP.
- **Performance** — Logging on every request must be fast. Avoid allocations in the hot path.

## Requirements

- [ ] JSON-formatted log output
- [ ] Log levels: debug, info, warn, error
- [ ] Configurable minimum log level
- [ ] Per-request fields: method, path, service, backend, status, latency_ms, client_ip
- [ ] Error logs include error details
- [ ] Structured logger injected via context or middleware

## Questions to Answer Before Coding

1. Why JSON logs instead of plain text?
2. How do you minimize allocations when logging on every request?
3. Should you use Go's `slog` package or a third-party logger?
4. How do you attach request-scoped fields to all log entries for that request?
