package waymark

import (
	"net/http"
)

// Router registers routes to be matched and dispatches a handler.
// It implements the http.Handler interface, so it can be registered
// to serve requests.
type Router struct {
	routes                  []*Route
	namedRoutes             map[string]*Route
	middlewares             []MiddlewareFunc
	notFoundHandler         http.Handler
	methodNotAllowedHandler http.Handler
	parent                  *Router
	parentRoute             *Route
	strictSlash             bool
	skipClean               bool
	useEncodedPath          bool
}

// NewRouter creates a new Router.
func NewRouter() *Router {
	return &Router{
		namedRoutes: make(map[string]*Route),
	}
}

// ServeHTTP dispatches the handler registered in the matched route.
func (router *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Will be fully implemented in Phase 4.
}
