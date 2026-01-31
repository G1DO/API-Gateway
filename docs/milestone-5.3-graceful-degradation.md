# Milestone 5.3: Graceful Degradation

**Phase:** 5 — Health Checking
**Status:** [x] Complete

## Goal

Automatically remove unhealthy backends from the load balancer pool and gradually reintroduce them when they recover.

## Key Concepts

- **Pool management** — Unhealthy backends are removed from the active pool. Healthy ones are added back.
- **Warm-up** — A recovering backend shouldn't immediately get full traffic. Gradually increase its share.
- **Total failure** — What happens when ALL backends are unhealthy? Options: return 503, try anyway, use stale responses.
- **Flap prevention** — Rapid healthy/unhealthy toggling causes instability. Require sustained health before reintroduction.

## Requirements

- [ ] Remove unhealthy backends from load balancer pool
- [ ] Re-add backends when health checks pass consistently
- [ ] Gradual warm-up: recovering backend gets reduced traffic initially
- [ ] Handle all-backends-down scenario (503 Service Unavailable)
- [ ] Prevent flapping with sustained-health requirements

## Questions to Answer Before Coding

1. How do you implement gradual warm-up in the load balancer?
2. What should the gateway return when all backends are down?
3. How many consecutive successes should be required before full reintroduction?
4. What's the risk of aggressive removal vs conservative removal?
