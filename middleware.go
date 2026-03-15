package waymark

import (
	"net/http"
	"strings"
)

// MiddlewareFunc is a function that wraps an http.Handler.
type MiddlewareFunc func(http.Handler) http.Handler

// CORSMethodMiddleware sets the Access-Control-Allow-Methods header based on
// the methods registered for the matched route. This is intended to be used
// as router-level middleware.
func CORSMethodMiddleware(r *Router) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			allMethods := getAllMethodsForPath(r, req)
			if len(allMethods) > 0 {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(allMethods, ","))
			}
			next.ServeHTTP(w, req)
		})
	}
}

// getAllMethodsForPath returns all HTTP methods registered for routes
// matching the given request path.
func getAllMethodsForPath(r *Router, req *http.Request) []string {
	seen := make(map[string]bool)
	for _, route := range r.routes {
		if route.pathPattern == nil {
			continue
		}
		if _, ok := route.pathPattern.match(req.URL.Path); ok {
			if len(route.methods) > 0 {
				for _, m := range route.methods {
					seen[m] = true
				}
			}
		}
	}

	// Always include OPTIONS if there are any methods.
	if len(seen) > 0 {
		seen["OPTIONS"] = true
	}

	methods := make([]string, 0, len(seen))
	for m := range seen {
		methods = append(methods, m)
	}
	return methods
}

// WalkFn is the type of the function called for each route visited by Walk.
type WalkFn func(route *Route, router *Router, ancestors []*Route) error

// Walk walks the router and all its sub-routers, calling walkFn for each route.
func Walk(r *Router, walkFn WalkFn) error {
	return walk(r, walkFn, nil)
}

// walk recursively traverses routes.
func walk(r *Router, walkFn WalkFn, ancestors []*Route) error {
	for _, route := range r.routes {
		if err := walkFn(route, r, ancestors); err != nil {
			return err
		}
		// If the route's handler is a subrouter, walk into it.
		if route.handler != nil {
			if subRouter, ok := route.handler.(*Router); ok {
				newAncestors := make([]*Route, len(ancestors)+1)
				copy(newAncestors, ancestors)
				newAncestors[len(ancestors)] = route
				if err := walk(subRouter, walkFn, newAncestors); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
