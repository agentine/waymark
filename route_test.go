package waymark

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRoutePathMatch(t *testing.T) {
	route := &Route{}
	route.Path("/users/{id}")
	route.Methods("GET")

	req := httptest.NewRequest("GET", "/users/42", nil)
	vars, ok, _ := route.match(req)
	if !ok {
		t.Fatal("expected match")
	}
	if vars["id"] != "42" {
		t.Errorf("vars[id] = %q, want %q", vars["id"], "42")
	}
}

func TestRouteMethodMismatch(t *testing.T) {
	route := &Route{}
	route.Path("/users/{id}").Methods("GET")

	req := httptest.NewRequest("POST", "/users/42", nil)
	_, ok, methodMismatch := route.match(req)
	if ok {
		t.Fatal("expected no match")
	}
	if !methodMismatch {
		t.Fatal("expected method mismatch signal")
	}
}

func TestRouteHeaderMatch(t *testing.T) {
	route := &Route{}
	route.Path("/api").Methods("POST").Headers("Content-Type", "application/json")

	req := httptest.NewRequest("POST", "/api", nil)
	req.Header.Set("Content-Type", "application/json")
	_, ok, _ := route.match(req)
	if !ok {
		t.Fatal("expected match")
	}

	req2 := httptest.NewRequest("POST", "/api", nil)
	req2.Header.Set("Content-Type", "text/plain")
	_, ok, _ = route.match(req2)
	if ok {
		t.Fatal("expected no match for wrong content-type")
	}
}

func TestRouteQueryMatch(t *testing.T) {
	route := &Route{}
	route.Path("/search").Queries("q", "{query}")

	req := httptest.NewRequest("GET", "/search?q=hello", nil)
	vars, ok, _ := route.match(req)
	if !ok {
		t.Fatal("expected match")
	}
	if vars["query"] != "hello" {
		t.Errorf("vars[query] = %q, want %q", vars["query"], "hello")
	}
}

func TestRouteQueryExactMatch(t *testing.T) {
	route := &Route{}
	route.Path("/search").Queries("type", "image")

	req := httptest.NewRequest("GET", "/search?type=image", nil)
	_, ok, _ := route.match(req)
	if !ok {
		t.Fatal("expected match")
	}

	req2 := httptest.NewRequest("GET", "/search?type=video", nil)
	_, ok, _ = route.match(req2)
	if ok {
		t.Fatal("expected no match for wrong query value")
	}
}

func TestRouteChaining(t *testing.T) {
	route := &Route{}
	result := route.Path("/users/{id}").Methods("GET", "POST").Headers("Accept", "application/json").Name("getUser")

	if result != route {
		t.Fatal("chaining should return same route")
	}
	if route.name != "getUser" {
		t.Errorf("name = %q, want %q", route.name, "getUser")
	}
}

func TestRouteURL(t *testing.T) {
	route := &Route{}
	route.Path("/users/{id}/posts/{pid}")

	u, err := route.URL("id", "42", "pid", "7")
	if err != nil {
		t.Fatal(err)
	}
	if u.Path != "/users/42/posts/7" {
		t.Errorf("URL path = %q, want %q", u.Path, "/users/42/posts/7")
	}
}

func TestRouteURLPath(t *testing.T) {
	route := &Route{}
	route.Path("/users/{id}")

	u, err := route.URLPath("id", "42")
	if err != nil {
		t.Fatal(err)
	}
	if u.Path != "/users/42" {
		t.Errorf("URLPath = %q, want %q", u.Path, "/users/42")
	}
}

func TestRouteURLWithHost(t *testing.T) {
	route := &Route{}
	route.Host("{subdomain}.example.com").Path("/users/{id}")

	u, err := route.URL("subdomain", "api", "id", "42")
	if err != nil {
		t.Fatal(err)
	}
	if u.Host != "api.example.com" {
		t.Errorf("Host = %q, want %q", u.Host, "api.example.com")
	}
	if u.Path != "/users/42" {
		t.Errorf("Path = %q, want %q", u.Path, "/users/42")
	}
}

func TestRouteGetters(t *testing.T) {
	route := &Route{}
	route.Path("/users/{id}").Methods("GET", "POST").Name("users")

	name := route.GetName()
	if name != "users" {
		t.Errorf("GetName() = %q, want %q", name, "users")
	}

	tpl, err := route.GetPathTemplate()
	if err != nil {
		t.Fatal(err)
	}
	if tpl != "/users/{id}" {
		t.Errorf("GetPathTemplate() = %q, want %q", tpl, "/users/{id}")
	}

	methods, err := route.GetMethods()
	if err != nil {
		t.Fatal(err)
	}
	if len(methods) != 2 || methods[0] != "GET" || methods[1] != "POST" {
		t.Errorf("GetMethods() = %v, want [GET POST]", methods)
	}

	regex, err := route.GetPathRegexp()
	if err != nil {
		t.Fatal(err)
	}
	if regex == "" {
		t.Error("GetPathRegexp() returned empty string")
	}
}

func TestRouteGettersErrors(t *testing.T) {
	route := &Route{}

	if _, err := route.GetPathTemplate(); err == nil {
		t.Error("expected error for missing path template")
	}
	if _, err := route.GetHostTemplate(); err == nil {
		t.Error("expected error for missing host template")
	}
	if _, err := route.GetMethods(); err == nil {
		t.Error("expected error for missing methods")
	}
	if _, err := route.GetPathRegexp(); err == nil {
		t.Error("expected error for missing path pattern")
	}
	if _, err := route.GetQueriesRegexp(); err == nil {
		t.Error("expected error for missing queries")
	}
	if _, err := route.GetQueriesTemplates(); err == nil {
		t.Error("expected error for missing queries templates")
	}
}

func TestVarsAndCurrentRoute(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)

	// Initially nil.
	if Vars(req) != nil {
		t.Error("expected nil vars on fresh request")
	}
	if CurrentRoute(req) != nil {
		t.Error("expected nil route on fresh request")
	}

	// Set and retrieve.
	vars := map[string]string{"id": "42"}
	route := &Route{name: "test"}
	req = setVars(req, vars)
	req = setCurrentRoute(req, route)

	got := Vars(req)
	if got["id"] != "42" {
		t.Errorf("Vars[id] = %q, want %q", got["id"], "42")
	}
	if CurrentRoute(req) != route {
		t.Error("CurrentRoute returned wrong route")
	}
}

func TestRouteHandler(t *testing.T) {
	route := &Route{}
	called := false
	route.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	h := route.GetHandler()
	if h == nil {
		t.Fatal("handler is nil")
	}
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	if !called {
		t.Error("handler was not called")
	}
}
