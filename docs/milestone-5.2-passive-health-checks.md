# Milestone 5.2: Passive Health Checks

**Phase:** 5 — Health Checking
**Status:** [ ] Not started

## Goal

Infer backend health from real traffic responses instead of (or in addition to) dedicated health probes.

## Key Concepts

- **Passive detection** — Use real request outcomes (5xx, timeouts, connection errors) as health signals.
- **No extra traffic** — Unlike active checks, passive checks don't add load.
- **Faster detection** — Under load, passive checks detect failures immediately (on the failing request).
- **Complementary** — Best used alongside active checks. Active catches idle-backend failures; passive catches under-load failures.

## Requirements

- [ ] Track 5xx responses and connection errors per backend
- [ ] Mark unhealthy when error rate exceeds a threshold within a time window
- [ ] Combine with active health check signals
- [ ] Configurable error rate threshold and time window
- [ ] Don't double-count errors (one request = one signal)

## Questions to Answer Before Coding

1. Why can't you rely solely on passive health checks?
2. What types of errors should count as health signals?
3. How do you define "error rate" — percentage of recent requests or count within a window?
4. How do passive and active checks interact? Which takes priority?
