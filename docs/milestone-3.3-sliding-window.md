# Milestone 3.3: Sliding Window

**Phase:** 3 — Rate Limiting
**Status:** [ ] Not started

## Goal

Implement a sliding window rate limiter as an alternative to token bucket, providing more accurate rate counting.

## Key Concepts

- **Fixed window problem** — A client can send 2x the limit at the boundary of two windows.
- **Sliding window log** — Store timestamps of each request. Count how many fall within the window. Accurate but memory-heavy.
- **Sliding window counter** — Weighted combination of current and previous window counts. Memory-efficient approximation.

## Requirements

- [ ] Sliding window counter algorithm (weighted previous + current window)
- [ ] Configurable window size and max requests
- [ ] Same interface as token bucket (drop-in replacement)
- [ ] Memory-efficient (don't store individual timestamps at scale)

## Questions to Answer Before Coding

1. What's the boundary burst problem with fixed windows?
2. How does the sliding window counter approximate the true sliding window?
3. What's the trade-off between sliding window log vs counter?
4. When would you choose sliding window over token bucket?
