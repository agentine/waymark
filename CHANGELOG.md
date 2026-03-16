# Changelog

## v0.1.0 — 2026-03-15

Initial release. Drop-in replacement for gorilla/mux with full API compatibility.

### Features

- **Pattern compiler** with regex constraints and compiled pattern caching
- **Route** with path variables, method/header/query/scheme/host matchers, and URL building
- **Router** implementing `http.Handler` with subrouters, strict slash, path cleaning, not-found and method-not-allowed handlers
- **Middleware** chaining via `Use()` and `CORSMethodMiddleware` for CORS preflight
- **Walk** for route introspection
- **compat package** (`github.com/agentine/waymark/compat`) for seamless drop-in migration from gorilla/mux via import path change only

### Quality

- 92.1% test coverage
- 8 benchmarks
- CI across Go 1.22, 1.23, 1.24
- Zero external dependencies
