package waymark

import (
	"net/http"
	"net/url"
	"path"
	"strings"
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
	// Clean the path unless skipClean is set.
	if !router.skipClean {
		p := req.URL.Path
		if !router.useEncodedPath {
			p = cleanPath(p)
		}
		if p != req.URL.Path {
			req.URL.Path = p
		}
	}

	var handler http.Handler
	var matchedVars map[string]string
	var matchedRoute *Route
	methodNotAllowed := false

	for _, route := range router.routes {
		// For prefix routes that are subrouters, do prefix matching.
		if route.isPrefix && route.handler != nil {
			if subRouter, ok := route.handler.(*Router); ok {
				prefix, prefixVars, prefixOK := route.matchPrefix(req)
				if prefixOK {
					// Strip matched prefix and dispatch to subrouter.
					subReq := req.Clone(req.Context())
					subReq.URL = cloneURL(req.URL)
					subReq.URL.Path = strings.TrimPrefix(req.URL.Path, prefix)
					if subReq.URL.Path == "" {
						subReq.URL.Path = "/"
					}
					// Merge prefix vars into context.
					existingVars := Vars(req)
					allVars := make(map[string]string)
					for k, v := range existingVars {
						allVars[k] = v
					}
					for k, v := range prefixVars {
						allVars[k] = v
					}
					subReq = setVars(subReq, allVars)
					subRouter.ServeHTTP(w, subReq)
					return
				}
				continue
			}
		}

		vars, matched, methodMismatch := route.match(req)
		if matched {
			matchedVars = vars
			matchedRoute = route
			handler = route.handler
			break
		}
		if methodMismatch {
			methodNotAllowed = true
		}
	}

	// Strict slash: try the alternate path if no match found.
	if handler == nil && router.strictSlash && !methodNotAllowed {
		altPath := req.URL.Path
		if strings.HasSuffix(altPath, "/") && altPath != "/" {
			altPath = altPath[:len(altPath)-1]
		} else {
			altPath = altPath + "/"
		}

		altReq := req.Clone(req.Context())
		altReq.URL = cloneURL(req.URL)
		altReq.URL.Path = altPath

		for _, route := range router.routes {
			if route.isPrefix {
				continue
			}
			vars, matched, _ := route.match(altReq)
			if matched {
				// Redirect to the correct path.
				u := *req.URL
				u.Path = altPath
				http.Redirect(w, req, u.String(), http.StatusMovedPermanently)
				_ = vars
				return
			}
		}
	}

	if handler != nil {
		// Merge with any existing vars (e.g. from parent subrouter).
		existingVars := Vars(req)
		if existingVars != nil {
			for k, v := range existingVars {
				if _, ok := matchedVars[k]; !ok {
					matchedVars[k] = v
				}
			}
		}
		// Set context values.
		req = setVars(req, matchedVars)
		req = setCurrentRoute(req, matchedRoute)

		// Apply middleware chain.
		handler = router.applyMiddleware(handler)
		handler.ServeHTTP(w, req)
		return
	}

	if methodNotAllowed {
		if router.methodNotAllowedHandler != nil {
			router.applyMiddleware(router.methodNotAllowedHandler).ServeHTTP(w, req)
		} else {
			http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// 404.
	if router.notFoundHandler != nil {
		router.applyMiddleware(router.notFoundHandler).ServeHTTP(w, req)
	} else {
		http.NotFound(w, req)
	}
}

// applyMiddleware wraps handler with the router's middleware chain.
func (router *Router) applyMiddleware(handler http.Handler) http.Handler {
	for i := len(router.middlewares) - 1; i >= 0; i-- {
		handler = router.middlewares[i](handler)
	}
	return handler
}

// Handle registers a new route with a matcher for the URL path.
func (router *Router) Handle(pathTpl string, handler http.Handler) *Route {
	route := router.NewRoute()
	route.Path(pathTpl).Handler(handler)
	return route
}

// HandleFunc registers a new route with a matcher for the URL path.
func (router *Router) HandleFunc(pathTpl string, f func(http.ResponseWriter, *http.Request)) *Route {
	return router.Handle(pathTpl, http.HandlerFunc(f))
}

// Path registers a new route with a matcher for the URL path.
func (router *Router) Path(tpl string) *Route {
	return router.NewRoute().Path(tpl)
}

// PathPrefix registers a new route with a matcher for the URL path prefix.
func (router *Router) PathPrefix(tpl string) *Route {
	return router.NewRoute().PathPrefix(tpl)
}

// Host registers a new route with a matcher for the request host.
func (router *Router) Host(tpl string) *Route {
	return router.NewRoute().Host(tpl)
}

// Methods registers a new route with a matcher for HTTP methods.
func (router *Router) Methods(methods ...string) *Route {
	return router.NewRoute().Methods(methods...)
}

// NewRoute creates an empty route associated with this router.
func (router *Router) NewRoute() *Route {
	route := &Route{router: router}
	router.routes = append(router.routes, route)
	return route
}

// Subrouter creates a new child router.
func (router *Router) Subrouter() *Router {
	sub := &Router{
		namedRoutes: make(map[string]*Route),
		parent:      router,
	}
	return sub
}

// Use appends middleware to the router's middleware chain.
func (router *Router) Use(mwf ...MiddlewareFunc) {
	router.middlewares = append(router.middlewares, mwf...)
}

// StrictSlash sets strict slash behavior. When true, if the route path
// is "/path/", accessing "/path" will redirect to "/path/", and vice versa.
func (router *Router) StrictSlash(value bool) *Router {
	router.strictSlash = value
	return router
}

// SkipClean sets whether to skip path cleaning.
func (router *Router) SkipClean(value bool) *Router {
	router.skipClean = value
	return router
}

// UseEncodedPath tells the router to match the encoded original path.
func (router *Router) UseEncodedPath() *Router {
	router.useEncodedPath = true
	return router
}

// NotFoundHandler sets the handler called when no route matches.
func (router *Router) NotFoundHandler(handler http.Handler) {
	router.notFoundHandler = handler
}

// MethodNotAllowedHandler sets the handler called when method is not allowed.
func (router *Router) MethodNotAllowedHandler(handler http.Handler) {
	router.methodNotAllowedHandler = handler
}

// Get returns a route registered with the given name.
func (router *Router) Get(name string) *Route {
	return router.getNamedRoutes()[name]
}

// GetRoute returns a route registered with the given name (alias for Get).
func (router *Router) GetRoute(name string) *Route {
	return router.Get(name)
}

// getNamedRoutes returns the map of named routes, searching parent routers.
func (router *Router) getNamedRoutes() map[string]*Route {
	if router.namedRoutes == nil {
		router.namedRoutes = make(map[string]*Route)
	}
	return router.namedRoutes
}

// cloneURL makes a shallow copy of a URL.
func cloneURL(u *url.URL) *url.URL {
	u2 := *u
	return &u2
}

// cleanPath returns the canonical path for p, eliminating . and .. elements.
func cleanPath(p string) string {
	if p == "" {
		return "/"
	}
	if p[0] != '/' {
		p = "/" + p
	}
	np := path.Clean(p)
	// path.Clean removes trailing slashes, but we need to preserve a single one.
	if strings.HasSuffix(p, "/") && np != "/" {
		np += "/"
	}
	return np
}
