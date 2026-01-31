package router

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// --- Config Parsing ---

func TestParseConfigValid(t *testing.T) {
	yaml := `
routes:
  - path: /api/users
    backends:
      - http://localhost:8081
      - http://localhost:8082
  - path: /
    backends:
      - http://localhost:8080
`
	cfg, err := ParseConfig([]byte(yaml))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(cfg.Routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(cfg.Routes))
	}
	if cfg.Routes[0].Path != "/api/users" {
		t.Fatalf("expected /api/users, got %s", cfg.Routes[0].Path)
	}
	if len(cfg.Routes[0].Backends) != 2 {
		t.Fatalf("expected 2 backends, got %d", len(cfg.Routes[0].Backends))
	}
}

func TestParseConfigWithHeaders(t *testing.T) {
	yaml := `
routes:
  - path: /api
    headers:
      Host: api.example.com
      X-API-Version: v2
    backends:
      - http://localhost:8081
`
	cfg, err := ParseConfig([]byte(yaml))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(cfg.Routes[0].Headers) != 2 {
		t.Fatalf("expected 2 headers, got %d", len(cfg.Routes[0].Headers))
	}
}

func TestParseConfigRejectsEmpty(t *testing.T) {
	yaml := `routes: []`
	_, err := ParseConfig([]byte(yaml))
	if err == nil {
		t.Fatal("should reject empty routes")
	}
}

func TestParseConfigRejectsNoBackends(t *testing.T) {
	yaml := `
routes:
  - path: /api
    backends: []
`
	_, err := ParseConfig([]byte(yaml))
	if err == nil {
		t.Fatal("should reject route with no backends")
	}
}

func TestParseConfigRejectsEmptyPath(t *testing.T) {
	yaml := `
routes:
  - path: ""
    backends:
      - http://localhost:8080
`
	_, err := ParseConfig([]byte(yaml))
	if err == nil {
		t.Fatal("should reject empty path")
	}
}

// --- Path-Based Routing ---

func TestRouterMatchesLongestPrefix(t *testing.T) {
	cfg, _ := ParseConfig([]byte(`
routes:
  - path: /api/users
    backends: ["http://users:8080"]
  - path: /api
    backends: ["http://api:8080"]
  - path: /
    backends: ["http://default:8080"]
`))
	r := New(cfg)

	tests := []struct {
		path    string
		wantBackend string
	}{
		{"/api/users/123", "http://users:8080"},
		{"/api/orders/456", "http://api:8080"},
		{"/static/file.js", "http://default:8080"},
	}

	for _, tc := range tests {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		route := r.Match(req)
		if route == nil {
			t.Fatalf("path %s: expected match, got nil", tc.path)
		}
		if route.Backends[0] != tc.wantBackend {
			t.Errorf("path %s: expected %s, got %s", tc.path, tc.wantBackend, route.Backends[0])
		}
	}
}

func TestRouterWildcard(t *testing.T) {
	cfg, _ := ParseConfig([]byte(`
routes:
  - path: /api/users/*
    backends: ["http://users:8080"]
`))
	r := New(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/users/123/profile", nil)
	route := r.Match(req)
	if route == nil {
		t.Fatal("expected match for wildcard route")
	}
}

func TestRouterNoMatch(t *testing.T) {
	cfg, _ := ParseConfig([]byte(`
routes:
  - path: /api
    backends: ["http://api:8080"]
`))
	r := New(cfg)

	req := httptest.NewRequest(http.MethodGet, "/other/path", nil)
	route := r.Match(req)
	if route != nil {
		t.Fatal("expected nil for unmatched path")
	}
}

// --- Header-Based Routing ---

func TestRouterMatchesHeaders(t *testing.T) {
	cfg, _ := ParseConfig([]byte(`
routes:
  - path: /api
    headers:
      X-API-Version: v2
    backends: ["http://v2:8080"]
  - path: /api
    backends: ["http://v1:8080"]
`))
	r := New(cfg)

	// With header → v2
	req := httptest.NewRequest(http.MethodGet, "/api/endpoint", nil)
	req.Header.Set("X-API-Version", "v2")
	route := r.Match(req)
	if route.Backends[0] != "http://v2:8080" {
		t.Fatalf("expected v2 backend, got %s", route.Backends[0])
	}

	// Without header → v1 (fallback)
	req2 := httptest.NewRequest(http.MethodGet, "/api/endpoint", nil)
	route2 := r.Match(req2)
	if route2.Backends[0] != "http://v1:8080" {
		t.Fatalf("expected v1 backend, got %s", route2.Backends[0])
	}
}

func TestRouterHostHeader(t *testing.T) {
	cfg, _ := ParseConfig([]byte(`
routes:
  - path: /
    headers:
      Host: shop.example.com
    backends: ["http://shop:8080"]
  - path: /
    headers:
      Host: blog.example.com
    backends: ["http://blog:8080"]
`))
	r := New(cfg)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Host", "shop.example.com")
	route := r.Match(req)
	if route == nil || route.Backends[0] != "http://shop:8080" {
		t.Fatal("expected shop backend for shop.example.com")
	}
}

func TestRouterHeaderPresenceCheck(t *testing.T) {
	cfg, _ := ParseConfig([]byte(`
routes:
  - path: /api
    headers:
      X-Canary: "*"
    backends: ["http://canary:8080"]
  - path: /api
    backends: ["http://stable:8080"]
`))
	r := New(cfg)

	// With X-Canary header (any value)
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("X-Canary", "anything")
	route := r.Match(req)
	if route.Backends[0] != "http://canary:8080" {
		t.Fatalf("expected canary backend, got %s", route.Backends[0])
	}

	// Without header → stable
	req2 := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	route2 := r.Match(req2)
	if route2.Backends[0] != "http://stable:8080" {
		t.Fatalf("expected stable backend, got %s", route2.Backends[0])
	}
}

// --- Hot Reload ---

func TestHotReloaderInitialLoad(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	err := os.WriteFile(cfgPath, []byte(`
routes:
  - path: /api
    backends: ["http://localhost:8080"]
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	hr, err := NewHotReloader(cfgPath, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("failed to create reloader: %v", err)
	}
	defer hr.Close()

	r := hr.Router()
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	route := r.Match(req)
	if route == nil {
		t.Fatal("expected route match after initial load")
	}
}

func TestHotReloaderDetectsChange(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	err := os.WriteFile(cfgPath, []byte(`
routes:
  - path: /api
    backends: ["http://old-backend:8080"]
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	hr, err := NewHotReloader(cfgPath, 50*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer hr.Close()

	// Verify initial config
	r := hr.Router()
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	route := r.Match(req)
	if route.Backends[0] != "http://old-backend:8080" {
		t.Fatal("expected old backend")
	}

	// Wait a bit, then update config (ensure mod time changes)
	time.Sleep(100 * time.Millisecond)

	err = os.WriteFile(cfgPath, []byte(`
routes:
  - path: /api
    backends: ["http://new-backend:8080"]
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Wait for reload
	time.Sleep(200 * time.Millisecond)

	r2 := hr.Router()
	req2 := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	route2 := r2.Match(req2)
	if route2.Backends[0] != "http://new-backend:8080" {
		t.Fatalf("expected new backend after reload, got %s", route2.Backends[0])
	}
}

func TestHotReloaderRejectsInvalidConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	err := os.WriteFile(cfgPath, []byte(`
routes:
  - path: /api
    backends: ["http://good-backend:8080"]
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	hr, err := NewHotReloader(cfgPath, 50*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer hr.Close()

	time.Sleep(100 * time.Millisecond)

	// Write invalid config (no backends)
	err = os.WriteFile(cfgPath, []byte(`
routes:
  - path: /api
    backends: []
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Wait for reload attempt
	time.Sleep(200 * time.Millisecond)

	// Should still have old config
	r := hr.Router()
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	route := r.Match(req)
	if route.Backends[0] != "http://good-backend:8080" {
		t.Fatalf("should keep old config on invalid reload, got %s", route.Backends[0])
	}
}
