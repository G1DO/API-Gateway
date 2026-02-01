# API Gateway Codemap

**Last Updated:** 2026-02-01
**Language:** Go 1.25
**Module:** `github.com/G1D0/Api-Gateway`
**Entry Point:** `cmd/gateway/main.go`

## Package Dependency Graph

```
cmd/gateway/main.go
├── internal/proxy       (uses lb.Balancer)
├── internal/lb          (no internal deps)
│
├── internal/middleware   (uses ratelimit, circuitbreaker)
│   ├── internal/ratelimit
│   └── internal/circuitbreaker
│
├── internal/router      (no internal deps, uses gopkg.in/yaml.v3)
├── internal/server      (no internal deps)
├── internal/observe     (uses prometheus/client_golang)
│
└── internal/health      (no internal deps)
    └── pool.go uses health.CombinedChecker
```

## Packages

| Package | Files | Purpose | Key Types |
|---------|-------|---------|-----------|
| `proxy` | 2 | Reverse proxy with connection pooling | `proxy` (unexported, implements `http.Handler`) |
| `lb` | 5 | Load balancing strategies | `Balancer` interface, `RoundRobin`, `WeightedRoundRobin`, `LeastConnections`, `ConsistentHash` |
| `ratelimit` | 4 | Rate limiting algorithms | `TokenBucket`, `PerClient`, `SlidingWindow` |
| `circuitbreaker` | 3 | Circuit breaker pattern | `CircuitBreaker`, `PerBackend`, `State` |
| `health` | 5 | Backend health checking | `ActiveChecker`, `PassiveChecker`, `CombinedChecker`, `HealthyPool` |
| `router` | 4 | YAML config + path/header routing | `Router`, `HotReloader`, `GatewayConfig`, `Route` |
| `middleware` | 7 | HTTP middleware composition | `Middleware` type, `Chain`, `Logging`, `Tracing`, `RateLimit`, `CircuitBreaker`, `ResponseCapture` |
| `server` | 2 | Graceful shutdown HTTP server | `Server`, `Config` |
| `observe` | 4 | Prometheus metrics + slog logging + tracing | `Metrics`, `NewLogger`, `GenerateTraceID` |

## Concurrency Patterns Used

| Pattern | Where | Why |
|---------|-------|-----|
| `atomic.AddUint64` | `lb.RoundRobin.Next()` | Lock-free counter for round robin rotation |
| `atomic.Int64` | `lb.LeastConnections` | Lock-free active connection tracking |
| `atomic.Uint32` | `circuitbreaker.CircuitBreaker` | Lock-free state reads on hot path |
| `atomic.Value` | `router.HotReloader` | Lock-free router swap on config reload |
| `sync.Mutex` | `lb.WeightedRoundRobin`, `ratelimit.TokenBucket`, `ratelimit.SlidingWindow`, `circuitbreaker.CircuitBreaker` | Write coordination |
| `sync.RWMutex` | `ratelimit.PerClient`, `circuitbreaker.PerBackend`, `health.ActiveChecker`, `health.PassiveChecker`, `health.HealthyPool` | Read-heavy maps with rare writes |
| Double-checked locking | `ratelimit.PerClient.Allow()`, `circuitbreaker.PerBackend.get()`, `health.PassiveChecker.getOrCreate()` | Lazy map entry creation without holding write lock on fast path |
| Background goroutine | `ratelimit.PerClient.gc()`, `health.ActiveChecker.run()`, `router.HotReloader.watch()` | Periodic work (GC, probes, file polling) |

## Interface Contracts

```go
// lb.Balancer -- all load balancers implement this
type Balancer interface {
    Next() string
}

// middleware.Middleware -- standard Go middleware pattern
type Middleware func(http.Handler) http.Handler

// health.Status -- enum (StatusUnknown=0, StatusHealthy=1, StatusUnhealthy=2)
```

## Config Format (router)

```yaml
routes:
  - path: /api/users
    backends:
      - http://localhost:8081
      - http://localhost:8082
  - path: /api/orders
    headers:
      X-Version: "v2"
    backends:
      - http://localhost:8083
```

## Prometheus Metrics

| Metric | Type | Labels |
|--------|------|--------|
| `gateway_requests_total` | Counter | service, status, method |
| `gateway_request_duration_seconds` | Histogram | service |
| `gateway_backend_healthy` | Gauge | backend |
| `gateway_rate_limited_total` | Counter | client |
| `gateway_circuit_state` | Gauge | backend |
| `gateway_active_connections` | Gauge | backend |

## Current State

The individual packages are complete and tested. `cmd/gateway/main.go` currently wires up only `lb.RoundRobin` + `proxy` on `:9000`. The middleware chain, router, server with graceful shutdown, health checks, and observability are built but not yet composed in main.
