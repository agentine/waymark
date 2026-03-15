package waymark

import (
	"net/http"
	"testing"
)

func TestWalk(t *testing.T) {
	r := NewRouter()
	r.HandleFunc("/users", func(w http.ResponseWriter, req *http.Request) {}).Name("listUsers")
	r.HandleFunc("/users/{id}", func(w http.ResponseWriter, req *http.Request) {}).Name("getUser")

	var names []string
	err := Walk(r, func(route *Route, router *Router, ancestors []*Route) error {
		name := route.GetName()
		if name != "" {
			names = append(names, name)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 {
		t.Fatalf("got %d named routes, want 2", len(names))
	}
	if names[0] != "listUsers" || names[1] != "getUser" {
		t.Errorf("names = %v, want [listUsers, getUser]", names)
	}
}

func TestWalkSubrouters(t *testing.T) {
	r := NewRouter()
	r.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {}).Name("home")

	sub := r.PathPrefix("/api").Subrouter()
	sub.HandleFunc("/users", func(w http.ResponseWriter, req *http.Request) {}).Name("apiUsers")

	var visited []struct {
		name      string
		ancestors int
	}
	err := Walk(r, func(route *Route, router *Router, ancestors []*Route) error {
		visited = append(visited, struct {
			name      string
			ancestors int
		}{route.GetName(), len(ancestors)})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// Should visit: "home" (0 ancestors), "/api" prefix route (0 ancestors), "apiUsers" (1 ancestor)
	if len(visited) != 3 {
		t.Fatalf("visited %d routes, want 3; visited: %+v", len(visited), visited)
	}
	// The subrouter's route should have 1 ancestor.
	if visited[2].ancestors != 1 {
		t.Errorf("apiUsers ancestors = %d, want 1", visited[2].ancestors)
	}
}

func TestWalkError(t *testing.T) {
	r := NewRouter()
	r.HandleFunc("/test", func(w http.ResponseWriter, req *http.Request) {})

	testErr := http.ErrAbortHandler
	err := Walk(r, func(route *Route, router *Router, ancestors []*Route) error {
		return testErr
	})
	if err != testErr {
		t.Errorf("err = %v, want %v", err, testErr)
	}
}
