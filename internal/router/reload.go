package router

import (
	"context"
	"log"
	"os"
	"sync/atomic"
	"time"
)

// HotReloader watches a config file and atomically swaps the router
// when changes are detected.
//
// Uses polling (not fsnotify) for simplicity and cross-platform reliability.
// The active router is stored in atomic.Value for lock-free reads.
type HotReloader struct {
	configPath string
	interval   time.Duration
	router     atomic.Value  // stores *Router
	lastModTime time.Time
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewHotReloader creates a hot reloader that watches configPath and
// polls for changes every interval.
func NewHotReloader(configPath string, interval time.Duration) (*HotReloader, error) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(configPath)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	hr := &HotReloader{
		configPath:  configPath,
		interval:    interval,
		lastModTime: info.ModTime(),
		ctx:         ctx,
		cancel:      cancel,
	}

	hr.router.Store(New(cfg))

	go hr.watch()
	return hr, nil
}

// Router returns the current active router (lock-free read).
func (hr *HotReloader) Router() *Router {
	return hr.router.Load().(*Router)
}

// Close stops the file watcher.
func (hr *HotReloader) Close() {
	hr.cancel()
}

// watch polls the config file for changes.
func (hr *HotReloader) watch() {
	ticker := time.NewTicker(hr.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hr.checkAndReload()
		case <-hr.ctx.Done():
			return
		}
	}
}

// checkAndReload checks if the config file changed and reloads if so.
func (hr *HotReloader) checkAndReload() {
	info, err := os.Stat(hr.configPath)
	if err != nil {
		log.Printf("hot reload: cannot stat config: %v", err)
		return
	}

	if !info.ModTime().After(hr.lastModTime) {
		return // no change
	}

	log.Printf("hot reload: config file changed, reloading...")

	cfg, err := LoadConfig(hr.configPath)
	if err != nil {
		log.Printf("hot reload: invalid config, keeping old: %v", err)
		return // keep running with old config
	}

	newRouter := New(cfg)
	hr.router.Store(newRouter) // atomic swap
	hr.lastModTime = info.ModTime()

	log.Printf("hot reload: config reloaded successfully (%d routes)", len(cfg.Routes))
}
