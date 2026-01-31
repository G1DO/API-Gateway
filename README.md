# API Gateway

A production-grade API gateway built from scratch in Go. Not an nginx wrapper — implementing the core primitives (reverse proxy, load balancing, rate limiting, circuit breakers) to understand what happens between clients and backends.

## The Problem

```
Client ──HTTP──► Backend
```

Works until the backend overloads, crashes, or gets abused. You need rate limiting, load distribution, health checking, and failure handling. You could add these to every service, or put a gateway in front.

```
                    ┌─────────────────────────────────────┐
                    │           API GATEWAY               │
                    │                                     │
Client ──────────►  │  [Rate Limit] → [Route] → [LB]      │
                    │                             │       │
                    │              ┌──────────────┼────┐  │
                    │              ▼              ▼    ▼  │
                    │          Backend A    Backend B  C  │
                    └─────────────────────────────────────┘
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                          GATEWAY                            │
│                                                             │
│   Acceptor ──► Router ──► LB Pool                           │
│                              │                              │
│   Rate Limiter    Circuit Breaker ◄──┘                      │
│                        │                                    │
│                   Conn Pool                                 │
│                        │                                    │
└────────────────────────┼────────────────────────────────────┘
                         │
           ┌─────────────┼─────────────┐
           ▼             ▼             ▼
       Backend A     Backend B     Backend C
```

## Tech Stack

- **Language:** Go
- **HTTP:** `net/http` standard library
- **Configuration:** YAML (hot-reloadable)
- **Metrics:** Prometheus format
- **Logging:** Structured JSON

## Milestones

### Phase 1: Reverse Proxy

| Milestone | Description | Status |
|-----------|-------------|--------|
| [1.1 — Basic Proxy](docs/milestone-1.1-basic-proxy.md) | Forward requests to a single backend | [x] |
| [1.2 — Connection Pooling](docs/milestone-1.2-connection-pooling.md) | Reuse TCP connections to backends | [x] |
| [1.3 — Timeouts](docs/milestone-1.3-timeouts.md) | Connection, request, and idle timeouts | [x] |

### Phase 2: Load Balancing

| Milestone | Description | Status |
|-----------|-------------|--------|
| [2.1 — Round Robin](docs/milestone-2.1-round-robin.md) | Sequential rotation across backends | [x] |
| [2.2 — Weighted Round Robin](docs/milestone-2.2-weighted-round-robin.md) | Proportional traffic by backend weight | [x] |
| [2.3 — Least Connections](docs/milestone-2.3-least-connections.md) | Route to least-loaded backend | [x] |
| [2.4 — Consistent Hashing](docs/milestone-2.4-consistent-hashing.md) | Sticky sessions via hash ring | [x] |

### Phase 3: Rate Limiting

| Milestone | Description | Status |
|-----------|-------------|--------|
| [3.1 — Token Bucket](docs/milestone-3.1-token-bucket.md) | Token bucket rate limiting algorithm | [x] |
| [3.2 — Per-Client Limiting](docs/milestone-3.2-per-client-limiting.md) | Separate limits per client | [x] |
| [3.3 — Sliding Window](docs/milestone-3.3-sliding-window.md) | Sliding window alternative | [x] |

### Phase 4: Circuit Breaker

| Milestone | Description | Status |
|-----------|-------------|--------|
| [4.1 — State Machine](docs/milestone-4.1-circuit-breaker-state-machine.md) | Closed/Open/Half-Open circuit breaker | [x] |
| [4.2 — Per-Backend Circuits](docs/milestone-4.2-per-backend-circuits.md) | Isolated circuit per backend | [x] |

### Phase 5: Health Checking

| Milestone | Description | Status |
|-----------|-------------|--------|
| [5.1 — Active Health Checks](docs/milestone-5.1-active-health-checks.md) | Periodic backend probing | [x] |
| [5.2 — Passive Health Checks](docs/milestone-5.2-passive-health-checks.md) | Infer health from real traffic | [x] |
| [5.3 — Graceful Degradation](docs/milestone-5.3-graceful-degradation.md) | Auto-remove/reintroduce backends | [x] |

### Phase 6: Routing & Configuration

| Milestone | Description | Status |
|-----------|-------------|--------|
| [6.1 — Path-Based Routing](docs/milestone-6.1-path-based-routing.md) | Route by URL path | [x] |
| [6.2 — Header-Based Routing](docs/milestone-6.2-header-based-routing.md) | Route by Host/custom headers | [x] |
| [6.3 — Hot Reload](docs/milestone-6.3-hot-reload.md) | Apply config changes without restart | [x] |

### Phase 7: Observability

| Milestone | Description | Status |
|-----------|-------------|--------|
| [7.1 — Metrics](docs/milestone-7.1-metrics.md) | Prometheus metrics endpoint | [ ] |
| [7.2 — Structured Logging](docs/milestone-7.2-structured-logging.md) | JSON-formatted request logs | [ ] |
| [7.3 — Request Tracing](docs/milestone-7.3-request-tracing.md) | Trace ID propagation | [ ] |

## Project Structure

```
api/
├── cmd/gateway/main.go              # Entry point
├── internal/
│   ├── proxy/
│   │   ├── proxy.go                 # Reverse proxy with connection pooling + timeouts
│   │   └── proxy_test.go
│   ├── lb/
│   │   ├── lb.go                    # Round-robin load balancer
│   │   ├── wrr.go                   # Smooth weighted round-robin
│   │   ├── leastconn.go             # Least-connections
│   │   ├── consistenthash.go        # Consistent hashing (hash ring)
│   │   └── lb_test.go
│   ├── ratelimit/
│   │   ├── tokenbucket.go           # Token bucket algorithm
│   │   ├── perclient.go             # Per-client rate limiter with eviction
│   │   ├── slidingwindow.go         # Sliding window counter
│   │   └── ratelimit_test.go
│   ├── circuitbreaker/
│   │   ├── circuitbreaker.go        # State machine (closed/open/half-open)
│   │   ├── perbackend.go            # Per-backend circuit breaker manager
│   │   └── circuitbreaker_test.go
│   ├── health/
│   │   ├── active.go                # Periodic health probing
│   │   ├── passive.go               # Infer health from traffic
│   │   ├── combined.go              # Active + passive combined
│   │   ├── pool.go                  # Backend pool management
│   │   └── health_test.go
│   └── router/
│       ├── config.go                # YAML config parsing + validation
│       ├── router.go                # Prefix-based routing with header matching
│       ├── reload.go                # Hot config reloader (polling + atomic swap)
│       └── router_test.go
├── docs/                            # Milestone documentation
├── go.mod
└── README.md
```

## Building & Running

```bash
# Build
go build -o gateway cmd/gateway/main.go

# Run (currently starts with hardcoded backends on :9000)
./gateway

# Run tests
go test ./...
```
