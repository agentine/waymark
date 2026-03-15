// Package compat provides gorilla/mux API-compatible type aliases and
// function wrappers, enabling migration via import path change only.
//
// Usage: replace
//
//	import "github.com/gorilla/mux"
//
// with
//
//	import mux "github.com/agentine/waymark/compat"
//
// and all existing gorilla/mux code continues to work.
package compat

import (
	"net/http"

	"github.com/agentine/waymark"
)

// Type aliases matching gorilla/mux exported types.
type Router = waymark.Router
type Route = waymark.Route
type MiddlewareFunc = waymark.MiddlewareFunc
type WalkFn = waymark.WalkFn

// NewRouter creates a new Router.
func NewRouter() *Router {
	return waymark.NewRouter()
}

// Vars returns the route variables for the current request.
func Vars(r *http.Request) map[string]string {
	return waymark.Vars(r)
}

// CurrentRoute returns the matched route for the current request.
func CurrentRoute(r *http.Request) *Route {
	return waymark.CurrentRoute(r)
}

// CORSMethodMiddleware returns a middleware that sets Access-Control-Allow-Methods.
func CORSMethodMiddleware(r *Router) MiddlewareFunc {
	return waymark.CORSMethodMiddleware(r)
}

// Walk walks the router and all its sub-routers, calling walkFn for each route.
func Walk(r *Router, walkFn WalkFn) error {
	return waymark.Walk(r, walkFn)
}
