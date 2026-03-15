// Package waymark is a drop-in replacement for gorilla/mux, providing
// an HTTP request router and dispatcher with gorilla/mux API compatibility.
//
// It supports path variables with regex constraints, host-based routing,
// middleware chaining, subrouters, URL building, and route walking.
//
// Usage:
//
//	r := waymark.NewRouter()
//	r.HandleFunc("/articles/{category}/{id:[0-9]+}", ArticleHandler)
//	r.HandleFunc("/", HomeHandler)
//	http.ListenAndServe(":8080", r)
//
// Variables can be extracted from the request:
//
//	vars := waymark.Vars(req)
//	category := vars["category"]
//	id := vars["id"]
package waymark
