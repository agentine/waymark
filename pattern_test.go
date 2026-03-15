package waymark

import (
	"sync"
	"testing"
)

func TestParseTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		wantVars []string
		wantErr  bool
	}{
		{"static path", "/api/v1/users", nil, false},
		{"single variable", "/users/{id}", []string{"id"}, false},
		{"two variables", "/users/{id}/posts/{postID}", []string{"id", "postID"}, false},
		{"regex constraint", "/users/{id:[0-9]+}", []string{"id"}, false},
		{"nested braces in regex", "/users/{id:[0-9]{3}}", []string{"id"}, false},
		{"root path", "/", nil, false},
		{"empty variable", "/users/{}", nil, true},
		{"unclosed brace", "/users/{id", nil, true},
		{"consecutive variables", "/{a}{b}", []string{"a", "b"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segments, err := parseTemplate(tt.template)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseTemplate(%q) error = %v, wantErr %v", tt.template, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			var gotVars []string
			for _, seg := range segments {
				if seg.variable != "" {
					gotVars = append(gotVars, seg.variable)
				}
			}
			if len(gotVars) != len(tt.wantVars) {
				t.Fatalf("got vars %v, want %v", gotVars, tt.wantVars)
			}
			for i := range gotVars {
				if gotVars[i] != tt.wantVars[i] {
					t.Errorf("var[%d] = %q, want %q", i, gotVars[i], tt.wantVars[i])
				}
			}
		})
	}
}

func TestCompilePatternMatch(t *testing.T) {
	tests := []struct {
		name     string
		template string
		prefix   bool
		path     string
		wantOK   bool
		wantVars map[string]string
	}{
		{"exact static", "/api/v1", false, "/api/v1", true, map[string]string{}},
		{"static no match", "/api/v1", false, "/api/v2", false, nil},
		{"single var", "/users/{id}", false, "/users/42", true, map[string]string{"id": "42"}},
		{"single var no match", "/users/{id}", false, "/users/", false, nil},
		{"two vars", "/users/{id}/posts/{pid}", false, "/users/5/posts/hello", true, map[string]string{"id": "5", "pid": "hello"}},
		{"regex constraint match", "/users/{id:[0-9]+}", false, "/users/123", true, map[string]string{"id": "123"}},
		{"regex constraint no match", "/users/{id:[0-9]+}", false, "/users/abc", false, nil},
		{"nested regex", "/items/{code:[A-Z]{3}}", false, "/items/ABC", true, map[string]string{"code": "ABC"}},
		{"nested regex no match", "/items/{code:[A-Z]{3}}", false, "/items/AB", false, nil},
		{"prefix match", "/api", true, "/api/v1/users", true, map[string]string{}},
		{"prefix with var", "/api/{version}", true, "/api/v1/users", true, map[string]string{"version": "v1"}},
		{"root", "/", false, "/", true, map[string]string{}},
		{"trailing slash match", "/users/", false, "/users/", true, map[string]string{}},
		{"no trailing slash", "/users", false, "/users/", false, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp, err := compilePattern(tt.template, tt.prefix)
			if err != nil {
				t.Fatalf("compilePattern(%q) error: %v", tt.template, err)
			}

			vars, ok := cp.match(tt.path)
			if ok != tt.wantOK {
				t.Fatalf("match(%q) = _, %v; want %v", tt.path, ok, tt.wantOK)
			}
			if !ok {
				return
			}
			for k, want := range tt.wantVars {
				if got := vars[k]; got != want {
					t.Errorf("vars[%q] = %q, want %q", k, got, want)
				}
			}
		})
	}
}

func TestCompilePatternBuildPath(t *testing.T) {
	tests := []struct {
		name     string
		template string
		pairs    map[string]string
		want     string
		wantErr  bool
	}{
		{"static", "/api/v1", map[string]string{}, "/api/v1", false},
		{"single var", "/users/{id}", map[string]string{"id": "42"}, "/users/42", false},
		{"two vars", "/users/{id}/posts/{pid}", map[string]string{"id": "5", "pid": "10"}, "/users/5/posts/10", false},
		{"missing var", "/users/{id}", map[string]string{}, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp, err := compilePattern(tt.template, false)
			if err != nil {
				t.Fatalf("compilePattern error: %v", err)
			}
			got, err := cp.buildPath(tt.pairs)
			if (err != nil) != tt.wantErr {
				t.Fatalf("buildPath() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("buildPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPatternCache(t *testing.T) {
	// Clear cache for isolated test.
	patternCache = sync.Map{}

	p1, err := compilePattern("/users/{id}", false)
	if err != nil {
		t.Fatal(err)
	}
	p2, err := compilePattern("/users/{id}", false)
	if err != nil {
		t.Fatal(err)
	}
	if p1 != p2 {
		t.Error("expected cached pattern to be the same pointer")
	}

	// Prefix variant should be separate.
	p3, err := compilePattern("/users/{id}", true)
	if err != nil {
		t.Fatal(err)
	}
	if p1 == p3 {
		t.Error("prefix and non-prefix should be different cache entries")
	}
}

func TestMatchedPrefix(t *testing.T) {
	cp, err := compilePattern("/api/{version}", true)
	if err != nil {
		t.Fatal(err)
	}

	prefix, vars, ok := cp.matchedPrefix("/api/v2/users/123")
	if !ok {
		t.Fatal("expected match")
	}
	if prefix != "/api/v2" {
		t.Errorf("prefix = %q, want %q", prefix, "/api/v2")
	}
	if vars["version"] != "v2" {
		t.Errorf("vars[version] = %q, want %q", vars["version"], "v2")
	}
}
