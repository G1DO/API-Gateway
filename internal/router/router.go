package router

import (
	"net/http"
	"sort"
	"strings"
)

// Route is a compiled route ready for matching.
type Route struct {
	Path     string            // prefix to match (e.g., "/api/users")
	Headers  map[string]string // headers that must match (all of them)
	Backends []string
}

// Router matches incoming requests to routes based on path and headers.
//
// Matching rules:
//  1. Path is matched by prefix (longest prefix wins)
//  2. If a route specifies headers, ALL must match
//  3. Routes with headers are checked before routes without (more specific first)
//  4. If no route matches, returns nil
type Router struct {
	routes []Route // sorted: longest path first, header routes before non-header routes
}

// New creates a router from config.
func New(cfg *GatewayConfig) *Router {
	routes := make([]Route, len(cfg.Routes))
	for i, rc := range cfg.Routes {
		// Strip trailing wildcard for prefix matching
		path := strings.TrimSuffix(rc.Path, "/*")
		path = strings.TrimSuffix(path, "*")

		routes[i] = Route{
			Path:     path,
			Headers:  rc.Headers,
			Backends: rc.Backends,
		}
	}

	// Sort by specificity:
	// 1. Longer paths first
	// 2. Routes with headers before routes without (at same path length)
	sort.Slice(routes, func(i, j int) bool {
		if len(routes[i].Path) != len(routes[j].Path) {
			return len(routes[i].Path) > len(routes[j].Path)
		}
		// Same length: routes with headers are more specific
		return len(routes[i].Headers) > len(routes[j].Headers)
	})

	return &Router{routes: routes}
}

// Match finds the best matching route for the request.
// Returns nil if no route matches.
func (r *Router) Match(req *http.Request) *Route {
	for i := range r.routes {
		route := &r.routes[i]

		// Check path prefix
		if !strings.HasPrefix(req.URL.Path, route.Path) {
			continue
		}

		// Check headers (all must match)
		if !matchHeaders(req, route.Headers) {
			continue
		}

		return route
	}
	return nil
}

// matchHeaders returns true if all required headers are present and match.
func matchHeaders(req *http.Request, required map[string]string) bool {
	for key, value := range required {
		got := req.Header.Get(key)
		if value == "*" {
			// Presence check: header must exist, any value
			if got == "" {
				return false
			}
		} else {
			// Exact match
			if got != value {
				return false
			}
		}
	}
	return true
}
