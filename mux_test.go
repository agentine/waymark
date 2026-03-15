package waymark

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBasicRouting(t *testing.T) {
	r := NewRouter()
	r.HandleFunc("/users/{id}", func(w http.ResponseWriter, req *http.Request) {
		vars := Vars(req)
		_, _ = w.Write([]byte("user:" + vars["id"]))
	})

	req := httptest.NewRequest("GET", "/users/42", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if rr.Body.String() != "user:42" {
		t.Errorf("body = %q, want %q", rr.Body.String(), "user:42")
	}
}

func TestMultipleRoutes(t *testing.T) {
	r := NewRouter()
	r.HandleFunc("/users", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("list"))
	})
	r.HandleFunc("/users/{id}", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("get:" + Vars(req)["id"]))
	})

	tests := []struct {
		path string
		want string
	}{
		{"/users", "list"},
		{"/users/42", "get:42"},
	}
	for _, tt := range tests {
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, httptest.NewRequest("GET", tt.path, nil))
		if rr.Body.String() != tt.want {
			t.Errorf("path %s: body = %q, want %q", tt.path, rr.Body.String(), tt.want)
		}
	}
}

func TestMethodRouting(t *testing.T) {
	r := NewRouter()
	r.HandleFunc("/api", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("GET"))
	}).Methods("GET")
	r.HandleFunc("/api", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("POST"))
	}).Methods("POST")

	// GET
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/api", nil))
	if rr.Body.String() != "GET" {
		t.Errorf("GET /api body = %q, want %q", rr.Body.String(), "GET")
	}

	// POST
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("POST", "/api", nil))
	if rr.Body.String() != "POST" {
		t.Errorf("POST /api body = %q, want %q", rr.Body.String(), "POST")
	}
}

func Test404(t *testing.T) {
	r := NewRouter()
	r.HandleFunc("/users", func(w http.ResponseWriter, req *http.Request) {})

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/nonexistent", nil))
	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestCustom404(t *testing.T) {
	r := NewRouter()
	r.NotFoundHandler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("custom 404"))
	}))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/nope", nil))
	if rr.Body.String() != "custom 404" {
		t.Errorf("body = %q, want %q", rr.Body.String(), "custom 404")
	}
}

func Test405(t *testing.T) {
	r := NewRouter()
	r.HandleFunc("/api", func(w http.ResponseWriter, req *http.Request) {}).Methods("GET")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("POST", "/api", nil))
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestCustom405(t *testing.T) {
	r := NewRouter()
	r.HandleFunc("/api", func(w http.ResponseWriter, req *http.Request) {}).Methods("GET")
	r.MethodNotAllowedHandler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = w.Write([]byte("custom 405"))
	}))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("POST", "/api", nil))
	if rr.Body.String() != "custom 405" {
		t.Errorf("body = %q, want %q", rr.Body.String(), "custom 405")
	}
}

func TestSubrouter(t *testing.T) {
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

func TestStrictSlash(t *testing.T) {
	r := NewRouter()
	r.StrictSlash(true)
	r.HandleFunc("/users/", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/users", nil))
	if rr.Code != http.StatusMovedPermanently {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusMovedPermanently)
	}
	loc := rr.Header().Get("Location")
	if loc != "/users/" {
		t.Errorf("Location = %q, want %q", loc, "/users/")
	}
}

func TestPathClean(t *testing.T) {
	r := NewRouter()
	r.HandleFunc("/api/users", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/api/../api/users", nil))
	if rr.Body.String() != "ok" {
		t.Errorf("body = %q, want %q", rr.Body.String(), "ok")
	}
}

func TestSkipClean(t *testing.T) {
	r := NewRouter()
	r.SkipClean(true)
	r.HandleFunc("/api/users", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	// With skip clean, the dirty path won't match.
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/api/../api/users", nil))
	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d (skip clean should not resolve path)", rr.Code, http.StatusNotFound)
	}
}

func TestNamedRoutes(t *testing.T) {
	r := NewRouter()
	r.HandleFunc("/users/{id}", func(w http.ResponseWriter, req *http.Request) {}).Name("getUser")

	route := r.Get("getUser")
	if route == nil {
		t.Fatal("named route not found")
	}
	u, err := route.URL("id", "42")
	if err != nil {
		t.Fatal(err)
	}
	if u.Path != "/users/42" {
		t.Errorf("URL path = %q, want %q", u.Path, "/users/42")
	}
}

func TestGetRoute(t *testing.T) {
	r := NewRouter()
	r.HandleFunc("/test", func(w http.ResponseWriter, req *http.Request) {}).Name("test")

	if r.GetRoute("test") == nil {
		t.Error("GetRoute returned nil for existing route")
	}
	if r.GetRoute("nonexistent") != nil {
		t.Error("GetRoute should return nil for missing route")
	}
}

func TestCurrentRouteInHandler(t *testing.T) {
	r := NewRouter()
	r.HandleFunc("/test", func(w http.ResponseWriter, req *http.Request) {
		route := CurrentRoute(req)
		if route == nil {
			t.Error("CurrentRoute returned nil inside handler")
			return
		}
		if route.GetName() != "myRoute" {
			t.Errorf("route name = %q, want %q", route.GetName(), "myRoute")
		}
	}).Name("myRoute")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/test", nil))
}

func TestRegexConstraints(t *testing.T) {
	r := NewRouter()
	r.HandleFunc("/users/{id:[0-9]+}", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("user:" + Vars(req)["id"]))
	})

	// Valid ID.
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/users/123", nil))
	if rr.Body.String() != "user:123" {
		t.Errorf("body = %q, want %q", rr.Body.String(), "user:123")
	}

	// Invalid ID (letters).
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/users/abc", nil))
	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d for non-numeric ID", rr.Code, http.StatusNotFound)
	}
}

func TestRouterMethods(t *testing.T) {
	r := NewRouter()
	route := r.Methods("GET", "POST")
	route.Path("/api").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/api", nil))
	if rr.Body.String() != "ok" {
		t.Errorf("body = %q, want %q", rr.Body.String(), "ok")
	}
}

func TestRouterHost(t *testing.T) {
	r := NewRouter()
	r.Host("{subdomain}.example.com").Path("/api").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("sub:" + Vars(req)["subdomain"]))
	})

	req := httptest.NewRequest("GET", "/api", nil)
	req.Host = "admin.example.com"
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Body.String() != "sub:admin" {
		t.Errorf("body = %q, want %q", rr.Body.String(), "sub:admin")
	}
}
