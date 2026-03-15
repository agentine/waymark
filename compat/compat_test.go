package compat

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestDropInCompatibility verifies that the compat package provides the same
// API surface as gorilla/mux, enabling migration via import path change only.
func TestDropInCompatibility(t *testing.T) {
	// This test uses the compat package exactly as gorilla/mux would be used.
	r := NewRouter()

	r.HandleFunc("/articles/{category}/{id:[0-9]+}", func(w http.ResponseWriter, req *http.Request) {
		vars := Vars(req)
		_, _ = w.Write([]byte(vars["category"] + ":" + vars["id"]))
	}).Methods("GET").Name("article")

	r.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("home"))
	})

	// Test basic routing.
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/articles/tech/42", nil))
	if rr.Body.String() != "tech:42" {
		t.Errorf("body = %q, want %q", rr.Body.String(), "tech:42")
	}

	// Test named route URL building.
	route := r.Get("article")
	if route == nil {
		t.Fatal("named route not found")
	}
	u, err := route.URL("category", "science", "id", "7")
	if err != nil {
		t.Fatal(err)
	}
	if u.Path != "/articles/science/7" {
		t.Errorf("URL path = %q, want %q", u.Path, "/articles/science/7")
	}
}

func TestCompatCurrentRoute(t *testing.T) {
	r := NewRouter()
	r.HandleFunc("/test", func(w http.ResponseWriter, req *http.Request) {
		route := CurrentRoute(req)
		if route == nil {
			t.Error("CurrentRoute returned nil")
		}
	}).Name("testRoute")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/test", nil))
}

func TestCompatMiddleware(t *testing.T) {
	r := NewRouter()
	r.HandleFunc("/api", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}).Methods("GET")

	r.Use(CORSMethodMiddleware(r))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/api", nil))
	if rr.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("CORS middleware not applied")
	}
}

func TestCompatWalk(t *testing.T) {
	r := NewRouter()
	r.HandleFunc("/a", func(w http.ResponseWriter, req *http.Request) {}).Name("routeA")
	r.HandleFunc("/b", func(w http.ResponseWriter, req *http.Request) {}).Name("routeB")

	count := 0
	err := Walk(r, func(route *Route, router *Router, ancestors []*Route) error {
		count++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("walked %d routes, want 2", count)
	}
}

func TestCompatSubrouter(t *testing.T) {
	r := NewRouter()
	sub := r.PathPrefix("/api").Subrouter()
	sub.HandleFunc("/users", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("users"))
	})

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/api/users", nil))
	if rr.Body.String() != "users" {
		t.Errorf("body = %q, want %q", rr.Body.String(), "users")
	}
}

func TestCompatStrictSlash(t *testing.T) {
	r := NewRouter()
	r.StrictSlash(true)
	r.HandleFunc("/users/", func(w http.ResponseWriter, req *http.Request) {})

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/users", nil))
	if rr.Code != http.StatusMovedPermanently {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusMovedPermanently)
	}
}
