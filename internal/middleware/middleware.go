package middleware

import "net/http"

// Middleware wraps an http.Handler with additional behavior.
// The standard Go middleware signature: takes a handler, returns a handler.
type Middleware func(http.Handler) http.Handler

// Chain composes multiple middleware into one. Middleware are applied
// in the order given: Chain(a, b, c)(handler) = a(b(c(handler))).
//
// This means the first middleware in the list is the outermost wrapper
// and runs first on the request path.
func Chain(middlewares ...Middleware) Middleware {
	return func(final http.Handler) http.Handler {
		// Apply in reverse so first middleware is outermost
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}
