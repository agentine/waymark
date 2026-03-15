package waymark

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMiddlewareChaining(t *testing.T) {
	r := NewRouter()

	var order []string
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			order = append(order, "mw1")
			next.ServeHTTP(w, req)
		})
	})
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			order = append(order, "mw2")
			next.ServeHTTP(w, req)
		})
	})
	r.HandleFunc("/test", func(w http.ResponseWriter, req *http.Request) {
		order = append(order, "handler")
	})

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/test", nil))

	want := "mw1,mw2,handler"
	got := strings.Join(order, ",")
	if got != want {
		t.Errorf("middleware order = %q, want %q", got, want)
	}
}

func TestMiddlewareOnNotFound(t *testing.T) {
	r := NewRouter()

	mwCalled := false
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			mwCalled = true
			next.ServeHTTP(w, req)
		})
	})

	r.NotFoundHandler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(404)
	}))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/nope", nil))

	if !mwCalled {
		t.Error("middleware should be called on not-found handler")
	}
}

func TestCORSMethodMiddleware(t *testing.T) {
	r := NewRouter()
	r.HandleFunc("/api", func(w http.ResponseWriter, req *http.Request) {}).Methods("GET")
	r.HandleFunc("/api", func(w http.ResponseWriter, req *http.Request) {}).Methods("POST")
	r.Use(CORSMethodMiddleware(r))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest("GET", "/api", nil))

	allow := rr.Header().Get("Access-Control-Allow-Methods")
	if allow == "" {
		t.Fatal("Access-Control-Allow-Methods header not set")
	}
	// Should contain GET, POST, and OPTIONS.
	for _, method := range []string{"GET", "POST", "OPTIONS"} {
		if !strings.Contains(allow, method) {
			t.Errorf("Access-Control-Allow-Methods %q missing %s", allow, method)
		}
	}
}
