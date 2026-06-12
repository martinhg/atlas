---
name: go-handler
description: "How to create a new API endpoint in Atlas (chi router, handler struct, pgxpool)"
metadata:
  keywords:
    - go
    - handler
    - chi
    - api
    - endpoint
    - http
    - rest
license: MIT
---

# Go Handler

## When to Use

Load this skill when:
- Adding a new HTTP endpoint to the Atlas backend
- Creating a new `internal/{domain}/handler.go`
- Wiring up routes in `cmd/atlas-server/main.go`
- Writing request parsing or JSON response helpers

## Rules

### Handler struct

```go
// internal/{domain}/handler.go
type Handler struct {
    store  *Store   // or an interface — see go-testing skill
    config *Config  // only if the handler needs config values
}

func NewHandler(store *Store) *Handler {
    return &Handler{store: store}
}
```

- Never inject `pgxpool.Pool` directly into a handler. Go through a `Store`.
- One handler struct per domain package (`auth`, `catalog`, `org`, …).
- Struct fields are unexported. Constructor is the only public entry point besides the methods.

### Method signature

All handler methods must match `http.HandlerFunc`:

```go
func (h *Handler) HandleListRepositories(w http.ResponseWriter, r *http.Request) {
```

Naming convention: `Handle{Verb}{Resource}` — e.g. `HandleCreateRepository`, `HandleGetRepository`.

### Request parsing

**JSON body:**
```go
var body struct {
    Name string `json:"name"`
}
if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
    jsonError(w, "invalid request body", http.StatusBadRequest)
    return
}
```

**URL param (chi):**
```go
id := chi.URLParam(r, "id")
```

**Query param:**
```go
q := r.URL.Query().Get("q")
```

### Response helpers

Use these two helpers consistently. Do NOT call `http.Error` for JSON endpoints.

```go
func jsonOK(w http.ResponseWriter, v any) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, msg string, status int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
```

Place helpers in `internal/{domain}/handler.go` (unexported, file-scoped) or in a shared `internal/platform/httputil` package if reused across domains.

### Logging

Use `log/slog` (already the project standard):

```go
slog.Error("failed to create repository", "error", err)
```

Log at the point of failure, not at the call site. Do not log and re-return the same error up the chain.

### Route registration in main.go

Add routes inside the existing `/api/v1` group:

```go
// cmd/atlas-server/main.go
repoStore   := catalog.NewStore(pool)
repoHandler := catalog.NewHandler(repoStore)

r.Route("/api/v1", func(r chi.Router) {
    // public routes
    r.Get("/repositories", repoHandler.HandleListRepositories)

    // protected routes — wrap in the auth middleware group
    r.Group(func(r chi.Router) {
        r.Use(auth.Middleware(cfg.JWTSecret))
        r.Post("/repositories", repoHandler.HandleCreateRepository)
        r.Get("/repositories/{id}", repoHandler.HandleGetRepository)
    })
})
```

### Accessing the authenticated user

```go
import "github.com/google/uuid"

userID, ok := r.Context().Value(auth.UserIDKey).(uuid.UUID)
if !ok {
    jsonError(w, "unauthorized", http.StatusUnauthorized)
    return
}
```

`auth.UserIDKey` is the typed context key set by `auth.Middleware`.

### Error status codes

| Situation | Status |
|-----------|--------|
| Missing / malformed body | 400 |
| Not authenticated | 401 |
| No permission | 403 |
| Resource not found | 404 |
| Conflict / duplicate | 409 |
| Internal / DB error | 500 |

### File layout

```
internal/
  {domain}/
    model.go       — structs, no logic
    store.go       — DB queries (pgxpool)
    handler.go     — HTTP handlers
    middleware.go  — optional, domain-specific middleware
```

## Example

Full minimal handler for a new domain:

```go
package catalog

import (
    "encoding/json"
    "log/slog"
    "net/http"
)

type Handler struct {
    store *Store
}

func NewHandler(store *Store) *Handler {
    return &Handler{store: store}
}

func (h *Handler) HandleListRepositories(w http.ResponseWriter, r *http.Request) {
    repos, err := h.store.ListRepositories(r.Context())
    if err != nil {
        slog.Error("failed to list repositories", "error", err)
        jsonError(w, "internal error", http.StatusInternalServerError)
        return
    }
    jsonOK(w, repos)
}

func jsonOK(w http.ResponseWriter, v any) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, msg string, status int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
```

Canonical reference: `internal/auth/handler.go`.
