# Milestone 2.4: Consistent Hashing

**Phase:** 2 — Load Balancing
**Status:** [x] Complete

## Goal

Hash a request attribute (client IP, header) to consistently map it to the same backend, with minimal disruption when backends are added or removed.

## Key Concepts

- **Hash ring** — Backends placed on a circular hash space. A request's hash finds the next backend clockwise.
- **Virtual nodes** — Each backend gets multiple points on the ring for better distribution.
- **Minimal disruption** — Adding/removing a backend only remaps ~1/N of keys (vs all keys with modulo hashing).
- **Sticky sessions** — Same client always hits same backend (useful for caching, stateful backends).

## Requirements

- [ ] Implement a hash ring data structure
- [ ] Place backends on the ring using consistent hashing
- [ ] Support virtual nodes for even distribution
- [ ] Hash by client IP (configurable to use other keys like headers)
- [ ] When a backend is removed, only its keys remap
- [ ] Implements the `Balancer` interface

## Questions to Answer Before Coding

1. Why does modulo hashing (`hash % N`) break when N changes?
2. How do virtual nodes improve distribution on the hash ring?
3. How many virtual nodes per backend is a good default?
4. What hash function should you use and why?
5. When would you choose consistent hashing over least connections?
