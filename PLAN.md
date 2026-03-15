# Waymark — Drop-in Replacement for gorilla/mux

## Overview

**Replaces:** [gorilla/mux](https://github.com/gorilla/mux) (21.9k stars, 98,604 importers, discontinued)
**Package:** `github.com/agentine/waymark`
**Language:** Go (minimum Go 1.22)

gorilla/mux is one of the most widely used HTTP routers in Go, with 98,604 known importers. After the Gorilla Toolkit was archived in Dec 2022 and briefly revived in July 2023, activity stalled again. The last commit was August 2024 (v1.8.0). It is marked **discontinued** on endoflife.date. There are 23 open issues and 12 unmerged PRs with no maintainer activity.

Alternatives like chi, gin, and Go 1.22+ stdlib require API changes. Waymark provides **gorilla/mux API compatibility** so existing projects can switch with a single import path change.

## Why Replace

- **No maintenance:** 1.5+ years since last commit, no security patches
- **Open CVE risk:** Unfixed issues accumulate without active triage
- **98,604 importers:** Massive blast radius for any future vulnerability
- **No drop-in alternative:** chi/gin/echo require code rewrites; Go 1.22+ stdlib covers only basic routing
- **Known unfixed bugs:** 23 open issues including routing edge cases

## Architecture

### Core Components

1. **Router** — Top-level request dispatcher implementing `http.Handler`
   - Route registration (`Handle`, `HandleFunc`, `Path`, `PathPrefix`)
   - Host-based routing (`Host`)
   - Subrouter support (`Subrouter`, `PathPrefix(...).Subrouter()`)
   - Middleware chaining (`Use`)
   - Walking routes (`Walk`)
   - Not-found and method-not-allowed handlers

2. **Route** — Individual route with matchers and handler
   - Path patterns with variables (`/users/{id}`)
   - Regex constraints (`/users/{id:[0-9]+}`)
   - Method matching (`Methods("GET", "POST")`)
   - Header matching (`Headers("Content-Type", "application/json")`)
   - Query parameter matching (`Queries("key", "value")`)
   - Scheme matching (`Schemes("https")`)
   - URL building (`URL`, `URLPath`)
   - Route naming (`Name`)

3. **Pattern Compiler** — Regex-based path pattern engine
   - Compile path templates to regex
   - Extract named variables
   - Support greedy and non-greedy segments
   - Cache compiled patterns for performance

4. **Middleware** — `func(http.Handler) http.Handler` compatible
   - Per-router middleware via `Use`
   - `CORSMethodMiddleware` for preflight response
   - Composable with stdlib and third-party middleware

5. **Compatibility Layer** — `github.com/agentine/waymark/compat`
   - Type aliases and adapter functions
   - `Vars(r)` function for extracting path variables
   - Enables migration via import path replacement only

### Key Design Decisions

- **net/http.ServeMux integration:** Use Go 1.22+ enhanced routing internally where possible for performance
- **gorilla/mux API surface:** Match all exported types, functions, and methods
- **`Vars(r *http.Request) map[string]string`:** Store variables in request context (same as gorilla/mux)
- **Concrete types:** Router and Route are concrete structs (matching gorilla/mux)
- **Zero dependencies:** No external dependencies

## Deliverables

1. Core router with full gorilla/mux API compatibility
2. Pattern compiler with regex constraints
3. Middleware support (`Use`, `CORSMethodMiddleware`)
4. Subrouter support
5. URL building from named routes
6. `Walk` function for route introspection
7. `compat` package for seamless drop-in migration
8. Comprehensive test suite (aim for >90% coverage)
9. Benchmark suite comparing against gorilla/mux and chi
10. Migration guide documentation

## Improvements Over gorilla/mux

- Active maintenance and security patches
- Go 1.22+ minimum (leveraging enhanced stdlib routing)
- Better pattern compilation caching
- Improved error messages for route conflicts
- `context.Context` patterns throughout
- Performance optimizations via compiled route tables
- Proper concurrent-safe route registration
