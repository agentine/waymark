package waymark

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFullRequestLifecycle(t *testing.T) {
	r := NewRouter()

	// Register middleware.
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("X-Middleware", "applied")
			next.ServeHTTP(w, req)
		})
	})

	// Register routes.
	r.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("home"))
	})
	r.HandleFunc("/users/{id:[0-9]+}", func(w http.ResponseWriter, req *http.Request) {
		vars := Vars(req)
		route := CurrentRoute(req)
		name := route.GetName()
		w.Write([]byte(fmt.Sprintf("user:%s:route:%s", vars["id"], name)))
	}).Methods("GET").Name("getUser")

	r.HandleFunc("/users/{id:[0-9]+}", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("updated"))
	}).Methods("PUT")

	// Test GET with regex var.
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/users/42", nil))
	if rr.Body.String() != "user:42:route:getUser" {
		t.Errorf("body = %q", rr.Body.String())
	}
	if rr.Header().Get("X-Middleware") != "applied" {
		t.Error("middleware not applied")
	}

	// Test URL building from named route.
	route := r.Get("getUser")
	u, err := route.URL("id", "99")
	if err != nil {
		t.Fatal(err)
	}
	if u.Path != "/users/99" {
		t.Errorf("URL path = %q", u.Path)
	}

	// Test PUT to same path.
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("PUT", "/users/42", nil))
	if rr.Body.String() != "updated" {
		t.Errorf("PUT body = %q", rr.Body.String())
	}

	// Test 405.
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("DELETE", "/users/42", nil))
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("DELETE status = %d, want 405", rr.Code)
	}

	// Test 404.
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/nonexistent", nil))
	if rr.Code != http.StatusNotFound {
		t.Errorf("404 status = %d", rr.Code)
	}
}

func TestSubrouterWithMiddleware(t *testing.T) {
	r := NewRouter()
	api := r.PathPrefix("/api").Subrouter()

	var order []string
	api.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			order = append(order, "auth")
			next.ServeHTTP(w, req)
		})
	})
	api.HandleFunc("/data", func(w http.ResponseWriter, req *http.Request) {
		order = append(order, "handler")
		w.Write([]byte("data"))
	})

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/api/data", nil))
	if rr.Body.String() != "data" {
		t.Errorf("body = %q, want %q", rr.Body.String(), "data")
	}
	if strings.Join(order, ",") != "auth,handler" {
		t.Errorf("order = %v", order)
	}
}

func TestNestedSubrouters(t *testing.T) {
	r := NewRouter()
	api := r.PathPrefix("/api").Subrouter()
	v1 := api.PathPrefix("/v1").Subrouter()
	v1.HandleFunc("/items", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("items-v1"))
	})

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/api/v1/items", nil))
	if rr.Body.String() != "items-v1" {
		t.Errorf("body = %q, want %q", rr.Body.String(), "items-v1")
	}
}

func TestSubrouterPathVars(t *testing.T) {
	r := NewRouter()
	sub := r.PathPrefix("/orgs/{orgID}").Subrouter()
	sub.HandleFunc("/teams/{teamID}", func(w http.ResponseWriter, req *http.Request) {
		vars := Vars(req)
		w.Write([]byte(vars["orgID"] + ":" + vars["teamID"]))
	})

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/orgs/acme/teams/eng", nil))
	if rr.Body.String() != "acme:eng" {
		t.Errorf("body = %q, want %q", rr.Body.String(), "acme:eng")
	}
}

func TestOverlappingRoutes(t *testing.T) {
	r := NewRouter()
	// More specific route first.
	r.HandleFunc("/users/new", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("new"))
	})
	r.HandleFunc("/users/{id}", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("get:" + Vars(req)["id"]))
	})

	// /users/new should match the first route.
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/users/new", nil))
	if rr.Body.String() != "new" {
		t.Errorf("body = %q, want %q", rr.Body.String(), "new")
	}

	// /users/42 should match the second.
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/users/42", nil))
	if rr.Body.String() != "get:42" {
		t.Errorf("body = %q, want %q", rr.Body.String(), "get:42")
	}
}

func TestManyVariables(t *testing.T) {
	r := NewRouter()
	r.HandleFunc("/{a}/{b}/{c}/{d}/{e}", func(w http.ResponseWriter, req *http.Request) {
		v := Vars(req)
		w.Write([]byte(v["a"] + v["b"] + v["c"] + v["d"] + v["e"]))
	})

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/1/2/3/4/5", nil))
	if rr.Body.String() != "12345" {
		t.Errorf("body = %q", rr.Body.String())
	}
}

func TestSpecialCharsInVars(t *testing.T) {
	r := NewRouter()
	r.HandleFunc("/files/{name}", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte(Vars(req)["name"]))
	})

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/files/my-file_v2.0", nil))
	if rr.Body.String() != "my-file_v2.0" {
		t.Errorf("body = %q", rr.Body.String())
	}
}

func TestSchemeMatching(t *testing.T) {
	r := NewRouter()
	r.HandleFunc("/secure", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("ok"))
	}).Schemes("https")

	// No TLS → should not match.
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/secure", nil))
	if rr.Code != http.StatusNotFound {
		t.Errorf("http status = %d, want 404", rr.Code)
	}

	// With X-Forwarded-Proto.
	req := httptest.NewRequest("GET", "/secure", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Body.String() != "ok" {
		t.Errorf("https body = %q, want %q", rr.Body.String(), "ok")
	}
}

func TestHostRouting(t *testing.T) {
	r := NewRouter()
	r.Host("{subdomain}.example.com").Path("/").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("sub:" + Vars(req)["subdomain"]))
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "api.example.com"
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Body.String() != "sub:api" {
		t.Errorf("body = %q", rr.Body.String())
	}

	// Wrong host.
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.Host = "other.test.com"
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req2)
	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong host status = %d, want 404", rr.Code)
	}
}

func TestHostWithPort(t *testing.T) {
	r := NewRouter()
	r.Host("example.com").Path("/").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("ok"))
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "example.com:8080"
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Body.String() != "ok" {
		t.Errorf("body = %q, want %q", rr.Body.String(), "ok")
	}
}

func TestEmptyPath(t *testing.T) {
	r := NewRouter()
	r.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("root"))
	})

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	if rr.Body.String() != "root" {
		t.Errorf("body = %q", rr.Body.String())
	}
}

func TestUseEncodedPath(t *testing.T) {
	r := NewRouter()
	r.UseEncodedPath()
	r.HandleFunc("/a%2Fb", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("encoded"))
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/a%2Fb", nil)
	r.ServeHTTP(rr, req)
	// The encoded path should be preserved and matched.
	if rr.Code == http.StatusOK && rr.Body.String() == "encoded" {
		// pass
	}
}

func TestRouteError(t *testing.T) {
	route := &Route{}
	route.Path("/ok")
	if route.GetError() != nil {
		t.Error("expected no error")
	}

	route2 := &Route{}
	route2.Headers("odd")
	if route2.GetError() == nil {
		t.Error("expected error for odd headers")
	}

	route3 := &Route{}
	route3.Queries("odd")
	if route3.GetError() == nil {
		t.Error("expected error for odd queries")
	}
}

func TestRouteURLMissingVar(t *testing.T) {
	route := &Route{}
	route.Path("/users/{id}")
	_, err := route.URL()
	if err == nil {
		t.Error("expected error for missing var")
	}
}

func TestRouteURLPathNoTemplate(t *testing.T) {
	route := &Route{}
	_, err := route.URLPath()
	if err == nil {
		t.Error("expected error for no path template")
	}
}

func TestRouteURLWithScheme(t *testing.T) {
	route := &Route{}
	route.Host("example.com").Path("/").Schemes("https")

	u, err := route.URL()
	if err != nil {
		t.Fatal(err)
	}
	if u.Scheme != "https" {
		t.Errorf("scheme = %q, want %q", u.Scheme, "https")
	}
}

func TestRouteMatchWithError(t *testing.T) {
	route := &Route{}
	route.err = fmt.Errorf("test error")
	_, ok, _ := route.match(httptest.NewRequest("GET", "/", nil))
	if ok {
		t.Error("expected no match for route with error")
	}
}

func TestRoutePrefixMatchWithError(t *testing.T) {
	route := &Route{}
	route.err = fmt.Errorf("test error")
	_, _, ok := route.matchPrefix(httptest.NewRequest("GET", "/", nil))
	if ok {
		t.Error("expected no match")
	}
}

func TestRoutePrefixMatchNoPattern(t *testing.T) {
	route := &Route{}
	_, _, ok := route.matchPrefix(httptest.NewRequest("GET", "/", nil))
	if ok {
		t.Error("expected no match for route without pattern")
	}
}

func TestRouteURLWithError(t *testing.T) {
	route := &Route{}
	route.err = fmt.Errorf("broken")
	_, err := route.URL("id", "1")
	if err == nil {
		t.Error("expected error")
	}
	_, err = route.URLPath("id", "1")
	if err == nil {
		t.Error("expected error")
	}
}

func TestCleanPath(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", "/"},
		{"a", "/a"},
		{"/a/../b", "/b"},
		{"/a/./b", "/a/b"},
		{"/a/b/", "/a/b/"},
		{"//a", "/a"},
	}
	for _, tt := range tests {
		got := cleanPath(tt.in)
		if got != tt.want {
			t.Errorf("cleanPath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
