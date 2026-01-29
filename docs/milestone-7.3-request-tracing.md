# Milestone 7.3: Request Tracing

**Phase:** 7 — Observability
**Status:** [ ] Not started

## Goal

Assign a unique trace ID to each request and propagate it to backends, enabling end-to-end request correlation across services.

## Key Concepts

- **Trace ID** — A unique identifier (UUID or similar) assigned to each incoming request.
- **Propagation** — Forward the trace ID to backends via a header (e.g., `X-Request-ID` or `X-Trace-ID`).
- **Correlation** — Logs from gateway and backends can be joined by trace ID.
- **Existing IDs** — If the client already sends a trace ID, use it instead of generating a new one.

## Requirements

- [ ] Generate a trace ID for each request if not already present
- [ ] Read existing trace ID from `X-Request-ID` header if provided
- [ ] Forward trace ID to backend via header
- [ ] Include trace ID in all log entries for that request
- [ ] Include trace ID in response headers back to client
- [ ] Use a fast ID generation method (UUID v4 or similar)

## Questions to Answer Before Coding

1. Why reuse an existing trace ID from the client instead of always generating a new one?
2. What's the standard header name for trace IDs? Is there a convention?
3. How do you make the trace ID available throughout the request lifecycle in Go?
4. What are the performance implications of UUID generation on every request?
