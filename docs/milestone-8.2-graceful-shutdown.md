# Milestone 8.2: Graceful Shutdown with Connection Draining

**Phase:** 8 — Production Readiness
**Status:** [ ] Not started

## Goal

When the gateway receives SIGTERM (or SIGINT), stop accepting new connections, let in-flight requests finish within a timeout, then exit cleanly. This prevents dropped requests during deployments and shows interviewers you think about production operations.

## Key Concepts

- **Signal handling** — Listen for `SIGTERM` and `SIGINT` using `os/signal.Notify`. These are what Kubernetes, Docker, and systemd send when stopping a process.
- **`http.Server.Shutdown(ctx)`** — Go's built-in graceful shutdown. Stops accepting new connections, waits for active requests to complete, respects the context deadline.
- **Drain timeout** — Don't wait forever for slow requests. Set a deadline (e.g., 30s). After that, force-close remaining connections.
- **Resource cleanup** — Close health checkers, stop hot reloader, flush metrics, close rate limiter GC goroutines.

## Requirements

- [ ] Replace `http.ListenAndServe` with `http.Server` struct for shutdown control
- [ ] Listen for SIGTERM and SIGINT signals
- [ ] Call `server.Shutdown(ctx)` with a configurable drain timeout
- [ ] Log shutdown progress: "shutting down...", "draining connections...", "shutdown complete"
- [ ] Close all background goroutines: health checkers, hot reloader, per-client GC
- [ ] Force-close after drain timeout expires
- [ ] Exit with code 0 on clean shutdown

## Questions to Answer Before Coding

1. What happens to in-flight requests when `Shutdown()` is called? Do they get cancelled or do they finish?
2. Why do you need a drain timeout? What happens if a request takes 10 minutes?
3. How does Kubernetes use SIGTERM and the graceful shutdown period during rolling deploys?
4. What's the difference between `Shutdown()` (graceful) and `Close()` (immediate)?
5. In what order should you clean up resources? (stop accepting → drain → close background → exit)
