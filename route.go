package waymark

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// contextKey is an unexported type for context keys in this package.
type contextKey int

const (
	varsKey         contextKey = iota
	routeKey
)

// Vars returns the route variables for the current request, if any.
func Vars(r *http.Request) map[string]string {
	if rv := r.Context().Value(varsKey); rv != nil {
		return rv.(map[string]string)
	}
	return nil
}

// CurrentRoute returns the matched route for the current request, if any.
func CurrentRoute(r *http.Request) *Route {
	if rv := r.Context().Value(routeKey); rv != nil {
		return rv.(*Route)
	}
	return nil
}

// setVars stores route variables in the request context.
func setVars(r *http.Request, vars map[string]string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), varsKey, vars))
}

// setCurrentRoute stores the matched route in the request context.
func setCurrentRoute(r *http.Request, route *Route) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), routeKey, route))
}

// Route stores information to match a request and build URLs.
type Route struct {
	handler     http.Handler
	pathPattern *compiledPattern
	hostPattern *compiledPattern
	pathTpl     string
	hostTpl     string
	methods     []string
	headers     []headerMatcher
	queries     []queryMatcher
	schemes     []string
	name        string
	err         error
	router      *Router
	isPrefix    bool
}

// headerMatcher matches a header key-value pair.
type headerMatcher struct {
	key   string
	value string
}

// queryMatcher matches a query parameter key-value pair.
type queryMatcher struct {
	key   string
	value string
}

// Handler sets a handler for the route.
func (r *Route) Handler(handler http.Handler) *Route {
	r.handler = handler
	return r
}

// HandlerFunc sets a handler function for the route.
func (r *Route) HandlerFunc(f func(http.ResponseWriter, *http.Request)) *Route {
	return r.Handler(http.HandlerFunc(f))
}

// Path sets the path template for the route.
func (r *Route) Path(tpl string) *Route {
	r.pathTpl = tpl
	r.isPrefix = false
	p, err := compilePattern(tpl, false)
	if err != nil {
		r.err = err
		return r
	}
	r.pathPattern = p
	return r
}

// PathPrefix sets a path prefix template for the route.
func (r *Route) PathPrefix(tpl string) *Route {
	r.pathTpl = tpl
	r.isPrefix = true
	p, err := compilePattern(tpl, true)
	if err != nil {
		r.err = err
		return r
	}
	r.pathPattern = p
	return r
}

// Host sets the host template for the route.
func (r *Route) Host(tpl string) *Route {
	r.hostTpl = tpl
	p, err := compilePattern(tpl, false)
	if err != nil {
		r.err = err
		return r
	}
	r.hostPattern = p
	return r
}

// Methods adds a method matcher to the route.
func (r *Route) Methods(methods ...string) *Route {
	for i, m := range methods {
		methods[i] = strings.ToUpper(m)
	}
	r.methods = methods
	return r
}

// Headers adds a header matcher to the route.
// Pairs must be key-value pairs: Headers("Content-Type", "application/json").
func (r *Route) Headers(pairs ...string) *Route {
	if len(pairs)%2 != 0 {
		r.err = fmt.Errorf("waymark: odd number of header matcher parameters")
		return r
	}
	for i := 0; i < len(pairs); i += 2 {
		r.headers = append(r.headers, headerMatcher{
			key:   http.CanonicalHeaderKey(pairs[i]),
			value: pairs[i+1],
		})
	}
	return r
}

// Queries adds a query parameter matcher to the route.
// Pairs must be key-value pairs: Queries("page", "{page}", "limit", "{limit}").
func (r *Route) Queries(pairs ...string) *Route {
	if len(pairs)%2 != 0 {
		r.err = fmt.Errorf("waymark: odd number of query matcher parameters")
		return r
	}
	for i := 0; i < len(pairs); i += 2 {
		r.queries = append(r.queries, queryMatcher{
			key:   pairs[i],
			value: pairs[i+1],
		})
	}
	return r
}

// Schemes adds a scheme matcher to the route.
func (r *Route) Schemes(schemes ...string) *Route {
	for i, s := range schemes {
		schemes[i] = strings.ToLower(s)
	}
	r.schemes = schemes
	return r
}

// Name sets the name for the route (used for URL building).
func (r *Route) Name(name string) *Route {
	r.name = name
	if r.router != nil {
		r.router.namedRoutes[name] = r
	}
	return r
}

// Subrouter creates a new router associated with this route as a subrouter.
func (r *Route) Subrouter() *Router {
	sub := &Router{
		namedRoutes: make(map[string]*Route),
		parent:      r.router,
		parentRoute: r,
	}
	r.handler = sub
	return sub
}

// GetName returns the name of the route.
func (r *Route) GetName() string {
	return r.name
}

// GetHandler returns the handler of the route.
func (r *Route) GetHandler() http.Handler {
	return r.handler
}

// GetPathTemplate returns the path template, or an error if none is set.
func (r *Route) GetPathTemplate() (string, error) {
	if r.pathTpl == "" {
		return "", fmt.Errorf("waymark: route has no path template")
	}
	return r.pathTpl, nil
}

// GetHostTemplate returns the host template, or an error if none is set.
func (r *Route) GetHostTemplate() (string, error) {
	if r.hostTpl == "" {
		return "", fmt.Errorf("waymark: route has no host template")
	}
	return r.hostTpl, nil
}

// GetMethods returns the methods the route matches, or an error if none are set.
func (r *Route) GetMethods() ([]string, error) {
	if len(r.methods) == 0 {
		return nil, fmt.Errorf("waymark: route has no methods")
	}
	methods := make([]string, len(r.methods))
	copy(methods, r.methods)
	return methods, nil
}

// GetPathRegexp returns the compiled path regex string, or an error.
func (r *Route) GetPathRegexp() (string, error) {
	if r.pathPattern == nil {
		return "", fmt.Errorf("waymark: route has no path pattern")
	}
	return r.pathPattern.regex.String(), nil
}

// GetQueriesRegexp returns the compiled query regexps, or an error.
func (r *Route) GetQueriesRegexp() ([]string, error) {
	if len(r.queries) == 0 {
		return nil, fmt.Errorf("waymark: route has no query matchers")
	}
	var result []string
	for _, q := range r.queries {
		result = append(result, q.key+"="+q.value)
	}
	return result, nil
}

// GetQueriesTemplates returns the query templates, or an error.
func (r *Route) GetQueriesTemplates() ([]string, error) {
	if len(r.queries) == 0 {
		return nil, fmt.Errorf("waymark: route has no query matchers")
	}
	var result []string
	for _, q := range r.queries {
		result = append(result, q.key+"="+q.value)
	}
	return result, nil
}

// GetError returns the route's compilation error, if any.
func (r *Route) GetError() error {
	return r.err
}

// URL builds a URL for the route using the given key-value pairs for variables.
func (r *Route) URL(pairs ...string) (*url.URL, error) {
	if r.err != nil {
		return nil, r.err
	}
	values := pairsToMap(pairs)

	var host, path string
	var err error

	if r.hostPattern != nil {
		host, err = r.hostPattern.buildPath(values)
		if err != nil {
			return nil, err
		}
	}

	if r.pathPattern != nil {
		path, err = r.pathPattern.buildPath(values)
		if err != nil {
			return nil, err
		}
	}

	u := &url.URL{
		Host: host,
		Path: path,
	}
	if host != "" {
		u.Scheme = "http"
		if len(r.schemes) > 0 {
			u.Scheme = r.schemes[0]
		}
	}
	return u, nil
}

// URLPath builds only the path portion of the URL.
func (r *Route) URLPath(pairs ...string) (*url.URL, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.pathPattern == nil {
		return nil, fmt.Errorf("waymark: route has no path template")
	}
	values := pairsToMap(pairs)
	path, err := r.pathPattern.buildPath(values)
	if err != nil {
		return nil, err
	}
	return &url.URL{Path: path}, nil
}

// match tests whether the request matches this route.
// It returns the extracted variables (path + host) and whether the match succeeded.
func (r *Route) match(req *http.Request) (map[string]string, bool, bool) {
	if r.err != nil {
		return nil, false, false
	}

	vars := make(map[string]string)
	methodMatch := true

	// Check methods.
	if len(r.methods) > 0 {
		methodMatch = false
		for _, m := range r.methods {
			if m == req.Method {
				methodMatch = true
				break
			}
		}
		if !methodMatch {
			// Method doesn't match, but path might — used for 405 detection.
		}
	}

	// Check path.
	if r.pathPattern != nil {
		pathVars, ok := r.pathPattern.match(req.URL.Path)
		if !ok {
			return nil, false, false
		}
		for k, v := range pathVars {
			vars[k] = v
		}
	}

	// If path matched but method didn't, signal method mismatch.
	if !methodMatch {
		return nil, false, true // pathMatch=false, methodMismatch=true
	}

	// Check host.
	if r.hostPattern != nil {
		host := req.Host
		if i := strings.IndexByte(host, ':'); i >= 0 {
			host = host[:i]
		}
		hostVars, ok := r.hostPattern.match(host)
		if !ok {
			return nil, false, false
		}
		for k, v := range hostVars {
			vars[k] = v
		}
	}

	// Check headers.
	for _, hm := range r.headers {
		if req.Header.Get(hm.key) != hm.value {
			return nil, false, false
		}
	}

	// Check query parameters.
	if len(r.queries) > 0 {
		q := req.URL.Query()
		for _, qm := range r.queries {
			val := q.Get(qm.key)
			// If the query value is a variable template, accept any value.
			if strings.HasPrefix(qm.value, "{") && strings.HasSuffix(qm.value, "}") {
				varName := qm.value[1 : len(qm.value)-1]
				vars[varName] = val
			} else if val != qm.value {
				return nil, false, false
			}
		}
	}

	// Check schemes.
	if len(r.schemes) > 0 {
		scheme := "http"
		if req.TLS != nil {
			scheme = "https"
		}
		if req.Header.Get("X-Forwarded-Proto") != "" {
			scheme = req.Header.Get("X-Forwarded-Proto")
		}
		matched := false
		for _, s := range r.schemes {
			if s == scheme {
				matched = true
				break
			}
		}
		if !matched {
			return nil, false, false
		}
	}

	return vars, true, false
}

// matchPrefix tests whether the request path matches this route's prefix.
func (r *Route) matchPrefix(req *http.Request) (prefix string, vars map[string]string, ok bool) {
	if r.err != nil || r.pathPattern == nil {
		return "", nil, false
	}
	return r.pathPattern.matchedPrefix(req.URL.Path)
}

// pairsToMap converts a variadic list of key-value pairs to a map.
func pairsToMap(pairs []string) map[string]string {
	m := make(map[string]string, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		m[pairs[i]] = pairs[i+1]
	}
	return m
}
