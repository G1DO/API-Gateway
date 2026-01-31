# Milestone 6.3: Hot Reload

**Phase:** 6 — Routing & Configuration
**Status:** [x] Complete

## Goal

Apply configuration changes without restarting the gateway. Watch the config file for changes and swap in the new configuration while gracefully draining existing connections.

## Key Concepts

- **File watching** — Detect config file changes using `fsnotify` or polling.
- **Atomic swap** — Replace the active config/router atomically so in-flight requests aren't corrupted.
- **Graceful drain** — Existing connections finish with the old config. New connections use the new config.
- **Validation** — Reject invalid config changes. Don't break a running gateway with a bad config.

## Requirements

- [x] Watch config file for changes (polling by mod time)
- [x] Parse and validate new config before applying
- [x] Atomically swap router/services to new config
- [ ] In-flight requests complete with their original routing (not yet implemented -- atomic swap is instant, no drain)
- [x] Log config reload events (success and failure)
- [x] Reject invalid config (keep running with old config)

## Implementation

- **File:** `internal/router/reload.go` -- `HotReloader` struct
- Uses polling (not fsnotify) -- checks `os.Stat` mod time at configurable interval
- `atomic.Value` stores `*Router` for lock-free reads via `Router()` method
- On change: `LoadConfig` -> validate -> `New(cfg)` -> `atomic.Store`
- Invalid config logged and skipped, old router stays active
- `context.Context` for clean shutdown via `Close()`

## Questions to Answer Before Coding

1. How do you atomically swap a config that multiple goroutines are reading?
2. What's the difference between `atomic.Value` and mutex for this?
3. How do you gracefully drain connections using the old config?
4. What should happen if the new config file is syntactically valid but semantically wrong?
5. Should you use fsnotify or polling? Trade-offs?
