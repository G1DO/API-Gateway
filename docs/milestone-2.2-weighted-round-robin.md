# Milestone 2.2: Weighted Round Robin

**Phase:** 2 — Load Balancing
**Status:** [x] Complete

## Goal

Assign weights to backends so higher-capacity servers receive proportionally more traffic.

## Key Concepts

- **Weights** — A backend with weight 3 gets 3x the traffic of weight 1.
- **Smooth distribution** — Naive approach (A,A,A,B,B,C) creates bursts. Smooth weighted round robin spreads them out (A,B,A,C,A,B).
- **Nginx's algorithm** — The smooth weighted round robin algorithm used by nginx.

## Requirements

- [ ] Each backend has a configurable weight (positive integer)
- [ ] Traffic distributed proportionally to weights
- [ ] Smooth distribution (no bursts to a single backend)
- [ ] Implements the same `Balancer` interface as round robin
- [ ] Default weight of 1 if not specified

## Questions to Answer Before Coding

1. Why does naive weighted round robin (repeat N times) cause problems?
2. How does nginx's smooth weighted round robin algorithm work?
3. What weight values make sense in practice?
4. How do you test that distribution matches the configured weights?
