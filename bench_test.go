package waymark

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkRouterMatchSimple(b *testing.B) {
	r := NewRouter()
	r.HandleFunc("/users", func(w http.ResponseWriter, req *http.Request) {})
	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()

	for range b.N {
		r.ServeHTTP(w, req)
	}
}

func BenchmarkRouterMatchParameterized(b *testing.B) {
	r := NewRouter()
	r.HandleFunc("/users/{id}", func(w http.ResponseWriter, req *http.Request) {})
	req := httptest.NewRequest("GET", "/users/42", nil)
	w := httptest.NewRecorder()

	for range b.N {
		r.ServeHTTP(w, req)
	}
}

func BenchmarkRouterMatchRegex(b *testing.B) {
	r := NewRouter()
	r.HandleFunc("/users/{id:[0-9]+}/posts/{pid:[a-z]+}", func(w http.ResponseWriter, req *http.Request) {})
	req := httptest.NewRequest("GET", "/users/42/posts/hello", nil)
	w := httptest.NewRecorder()

	for range b.N {
		r.ServeHTTP(w, req)
	}
}

func BenchmarkRouterMatchManyRoutes(b *testing.B) {
	r := NewRouter()
	for i := range 100 {
		path := fmt.Sprintf("/route%d/{id}", i)
		r.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {})
	}
	// Match the last route (worst case).
	req := httptest.NewRequest("GET", "/route99/42", nil)
	w := httptest.NewRecorder()

	for range b.N {
		r.ServeHTTP(w, req)
	}
}

func BenchmarkPatternCompile(b *testing.B) {
	for range b.N {
		// Clear cache to force recompilation.
		patternCache.Delete("/users/{id:[0-9]+}/posts/{pid}")
		_, _ = compilePattern("/users/{id:[0-9]+}/posts/{pid}", false)
	}
}

func BenchmarkPatternCompileCached(b *testing.B) {
	// Pre-warm.
	_, _ = compilePattern("/users/{id:[0-9]+}/posts/{pid}", false)

	for range b.N {
		_, _ = compilePattern("/users/{id:[0-9]+}/posts/{pid}", false)
	}
}

func BenchmarkVarsExtraction(b *testing.B) {
	r := NewRouter()
	r.HandleFunc("/users/{id}/posts/{pid}/comments/{cid}", func(w http.ResponseWriter, req *http.Request) {
		_ = Vars(req)
	})
	req := httptest.NewRequest("GET", "/users/42/posts/7/comments/3", nil)
	w := httptest.NewRecorder()

	for range b.N {
		r.ServeHTTP(w, req)
	}
}

func BenchmarkSubrouterMatch(b *testing.B) {
	r := NewRouter()
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/users/{id}", func(w http.ResponseWriter, req *http.Request) {})
	req := httptest.NewRequest("GET", "/api/users/42", nil)
	w := httptest.NewRecorder()

	for range b.N {
		r.ServeHTTP(w, req)
	}
}
