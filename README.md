# API Gateway

A production-grade API gateway built from scratch in Go using only the standard library. No frameworks, no nginx -- every component (reverse proxy, load balancing, rate limiting, circuit breaking, health checking, routing, observability) implemented from first principles.

## Why Build This?

Every request between a client and your backend needs:
- **Traffic control** -- rate limiting to prevent abuse
- **Load distribution** -- spreading requests across backends
- **Fault tolerance** -- circuit breakers to stop cascading failures
- **Health awareness** -- detecting and routing around unhealthy backends
- **Observability** -- metrics, logs, and traces to understand what's happening

You can bolt these onto every microservice, or you can put a gateway in front. This project implements that gateway -- not by configuring someone else's tool, but by building each primitive from scratch.

## Architecture

```
                         ┌──────────────────────────────────────────────┐
                         │                API GATEWAY                   │
                         │                                              │
  Client ─── HTTP ────►  │  Rate Limiter ──► Router ──► Load Balancer   │
                         │                                     │        │
                         │                          Circuit Breaker     │
                         │                                     │        │
                         │                    Health Checker ◄──┘        │
                         │                                              │
                         │  ┌─────────────────────────────────────┐     │
                         │  │  Observability (Metrics/Logs/Traces) │     │
                         │  └─────────────────────────────────────┘     │
                         └──────────────┬───────────┬──────────────────┘
                                        │           │
                              ┌─────────┼─────────┐ │
                              ▼         ▼         ▼ ▼
                          Backend A  Backend B  Backend C
```

**Request flow:**
1. Rate limiter checks if the client is within their allowed request budget
2. Router matches the request path and headers to a backend group
3. Load balancer picks a specific backend using the configured strategy
4. Circuit breaker decides whether to allow or fast-fail the request
5. Health checker feeds status back so unhealthy backends are skipped
6. Observability records metrics, structured logs, and trace IDs throughout

## What's Inside

### Reverse Proxy (`internal/proxy`)

Forwards HTTP requests to backends with connection pooling. Strips hop-by-hop headers, copies request/response bodies, and returns 502 on backend failure.

- Connection pooling via `http.Transport` (100 idle conns, 90s idle timeout)
- 5s dial timeout, 30s request timeout via context
- Hop-by-hop header stripping (Connection, Keep-Alive, Proxy-Authenticate, etc.)

### Load Balancing (`internal/lb`)

Four strategies behind a single `Balancer` interface (`Next() string`):

| Strategy | How It Works | When to Use |
|----------|-------------|-------------|
| **Round Robin** | Sequential rotation with atomic counter | Equal backends, stateless requests |
| **Weighted Round Robin** | Nginx's smooth weighted algorithm -- spreads proportionally without bursting | Backends with different capacities |
| **Least Connections** | Tracks active connections per backend with `atomic.Int64`, picks lowest | Variable request durations |
| **Consistent Hashing** | CRC32 hash ring with virtual nodes, binary search lookup | Sticky sessions, cache affinity |

### Rate Limiting (`internal/ratelimit`)

Three algorithms to control traffic:

| Algorithm | How It Works | Trade-off |
|-----------|-------------|-----------|
| **Token Bucket** | Lazy refill -- calculates accrued tokens on each `Allow()` call instead of a background ticker | Allows bursts up to capacity, then enforces sustained rate |
| **Per-Client** | Separate token bucket per client key with background GC for stale buckets | Memory grows with unique clients, GC keeps it bounded |
| **Sliding Window** | Weighted combination of previous + current window counts, constant memory | Smoother than fixed windows, prevents double-burst at boundaries |

All return `(ok bool, retryAfter time.Duration)` -- the caller knows exactly when to retry.

### Circuit Breaker (`internal/circuitbreaker`)

Prevents cascading failures with a three-state machine:

```
         requests succeed
              ┌───┐
              ▼   │
 ┌────────────────────┐     max failures     ┌────────────┐
 │      CLOSED        │ ──────────────────►  │    OPEN     │
 │  (allow requests)  │                      │ (reject all)│
 └────────────────────┘                      └──────┬──────┘
              ▲                                     │
              │          timeout expires            │
              │                                     ▼
              │                            ┌──────────────┐
              └─────── success ──────────  │  HALF-OPEN   │
                                           │ (test 1 req) │
                        failure ──────────►└──────────────┘
                        (back to OPEN)
```

- Fast reads via `atomic.Uint32`, writes protected by mutex
- `PerBackend` manager: isolated circuit per backend address, lazy initialization with double-checked locking

### Health Checking (`internal/health`)

Two complementary approaches combined with AND logic:

- **Active** -- periodic HTTP probes to a configurable health endpoint. Tracks consecutive successes/failures to prevent flapping
- **Passive** -- infers health from real traffic using a sliding time window. Marks unhealthy when error rate exceeds threshold (with minimum request count)
- **Combined** -- backend is healthy only if both active AND passive agree. Active catches idle failures, passive catches under-load failures
- **Pool** -- filters unhealthy backends from the load balancer's selection. Supports both fail-open (return all if none healthy) and fail-closed (return error)

### Routing (`internal/router`)

Path and header-based request routing with hot reload:

- **Config** -- YAML parser with validation for route definitions (prefix paths, header matchers, backend lists)
- **Router** -- prefix matching sorted by specificity (longest path first, header routes before wildcard)
- **Hot Reload** -- polls config file for changes, parses new config, swaps router atomically via `atomic.Value`. Invalid configs are rejected -- previous router stays active

### Observability (`internal/observe`)

Production instrumentation with zero external dependencies beyond Prometheus client:

- **Metrics** -- 6 Prometheus metric types: request count, latency histogram (5ms-10s buckets), backend health, rate limit hits, circuit breaker state, active connections. Exposed on `/metrics`
- **Logging** -- structured JSON via `log/slog` with request-scoped context (method, path, client IP, trace ID). Logger stored in context for downstream access
- **Tracing** -- 128-bit hex trace IDs from `crypto/rand`, propagated via `X-Request-ID` header. Reuses client-provided IDs when present

### Middleware (`internal/middleware`)

Composable middleware chain with standard `func(http.Handler) http.Handler` signature:

- **Chain** -- composes N middleware in order: `Chain(a, b, c)(handler)` = `a(b(c(handler)))`
- **Tracing** -- generates/propagates `X-Request-ID`, stores in context
- **Logging** -- structured JSON request logs (method, path, status, latency, client IP, trace ID)
- **RateLimit** -- per-client token bucket, returns 429 with `Retry-After` header. Supports custom key extraction functions
- **CircuitBreaker** -- per-backend circuit breaking, returns 503 when open. Records success/failure based on response status
- **ResponseCapture** -- wraps `http.ResponseWriter` to capture status code and bytes written (used by logging and circuit breaker middleware)

### Server (`internal/server`)

HTTP server with graceful shutdown:

- Listens for SIGTERM/SIGINT
- Stops accepting new connections
- Drains in-flight requests (configurable timeout, default 30s)
- Closes registered background resources (health checkers, rate limiter GC, hot reloaders)

## Project Structure

```
api/
├── cmd/gateway/
│   └── main.go                        # Entry point (basic: round robin + proxy)
├── internal/
│   ├── proxy/
│   │   ├── proxy.go                   # Reverse proxy with connection pooling
│   │   └── proxy_test.go
│   ├── lb/
│   │   ├── lb.go                      # Balancer interface + round robin
│   │   ├── wrr.go                     # Smooth weighted round robin
│   │   ├── leastconn.go               # Least connections
│   │   ├── consistenthash.go          # Consistent hashing with virtual nodes
│   │   └── lb_test.go
│   ├── ratelimit/
│   │   ├── tokenbucket.go             # Token bucket (lazy refill)
│   │   ├── perclient.go              # Per-client limiter with GC
│   │   ├── slidingwindow.go           # Sliding window counter
│   │   └── ratelimit_test.go
│   ├── circuitbreaker/
│   │   ├── circuitbreaker.go          # State machine (closed/open/half-open)
│   │   ├── perbackend.go             # Per-backend circuit isolation
│   │   └── circuitbreaker_test.go
│   ├── health/
│   │   ├── active.go                  # Periodic probe-based health checks
│   │   ├── passive.go                 # Traffic-inferred health checks
│   │   ├── combined.go               # AND-logic combined checker
│   │   ├── pool.go                    # Healthy backend pool filtering
│   │   └── health_test.go
│   ├── router/
│   │   ├── config.go                  # YAML route config parser
│   │   ├── router.go                  # Prefix + header matching
│   │   ├── reload.go                  # Hot reload with atomic swap
│   │   └── router_test.go
│   ├── middleware/
│   │   ├── middleware.go              # Chain composition
│   │   ├── tracing.go                # Request ID generation + propagation
│   │   ├── logging.go                # Structured JSON request logging
│   │   ├── ratelimit.go              # Rate limiting middleware
│   │   ├── circuitbreaker.go         # Circuit breaker middleware
│   │   ├── responsewriter.go         # ResponseWriter wrapper for status capture
│   │   └── middleware_test.go
│   ├── server/
│   │   ├── server.go                  # Graceful shutdown server
│   │   └── server_test.go
│   └── observe/
│       ├── metrics.go                 # Prometheus metrics (6 metric types)
│       ├── logging.go                 # Structured JSON logging (slog)
│       ├── tracing.go                 # Request ID generation + propagation
│       └── observe_test.go
├── docs/                              # Milestone documentation (23 files)
├── gateway                            # Compiled binary
├── go.mod
└── go.sum
```

## Design Decisions

**Zero external dependencies (except Prometheus client)** -- every algorithm implemented from scratch using Go's standard library. This is intentional: the goal is understanding, not shipping fast.

**Interfaces over concrete types** -- `lb.Balancer` is a single-method interface (`Next() string`). Any load balancing strategy plugs in without changing the proxy.

**Atomic operations for hot paths** -- round robin uses `atomic.AddUint64`, circuit breaker reads state via `atomic.Uint32`, router swaps via `atomic.Value`. Mutexes only where writes need coordination.

**Lazy initialization everywhere** -- token buckets refill on-demand (no background ticker), per-client limiters create buckets on first request, per-backend circuit breakers create on first access. No work until needed.

**Double-checked locking** -- used in `PerClient`, `PerBackend`, and `PassiveChecker` for lazy map initialization. RLock fast path, then Lock + recheck for creation.

**Concurrency safety as a requirement, not an afterthought** -- every component is safe for concurrent use. Tests include concurrent access scenarios.

## Building & Running

```bash
# Build
go build -o gateway cmd/gateway/main.go

# Run tests
go test ./...

# Run
./gateway
```

Note: `cmd/gateway/main.go` currently has a basic setup (round robin LB + proxy on `:9000`). The middleware, server, router, and observability packages are built but not yet wired into main.

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.25 |
| HTTP | `net/http` standard library |
| Configuration | YAML via `gopkg.in/yaml.v3` (hot-reloadable) |
| Metrics | Prometheus client (`github.com/prometheus/client_golang`) |
| Logging | `log/slog` (structured JSON) |
| Tracing | `X-Request-ID` header propagation |
| External deps | Prometheus client library + YAML parser only |
