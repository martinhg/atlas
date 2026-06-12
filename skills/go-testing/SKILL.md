---
name: go-testing
description: "Go testing conventions for Atlas: table-driven tests, store mocking, httptest handlers"
metadata:
  keywords:
    - go
    - testing
    - test
    - httptest
    - mock
    - table-driven
    - unit
license: MIT
---

# Go Testing

## When to Use

Load this skill when:
- Writing tests for Atlas Go packages
- Mocking the store layer for handler tests
- Setting up table-driven tests for business logic
- Running or checking test coverage

## Rules

### Test file location

```
internal/{domain}/
  handler.go
  handler_test.go   ← same package (white-box), or append _test for black-box
  store.go
  store_test.go
```

Use `package {domain}` (white-box) for unit tests. Use `package {domain}_test` (black-box) only when testing the public surface in isolation.

### Naming

```
TestFunctionName_scenario
```

Examples:
- `TestHandleMe_returnsUser`
- `TestHandleMe_missingToken`
- `TestUpsertUser_createsNewUser`
- `TestUpsertUser_updatesExisting`

### Table-driven tests

```go
func TestHandleMe_scenarios(t *testing.T) {
    tests := []struct {
        name       string
        userID     uuid.UUID
        setupStore func(*mockStore)
        wantStatus int
    }{
        {
            name:       "authenticated user returns 200",
            userID:     uuid.New(),
            setupStore: func(m *mockStore) {
                m.user = &User{Login: "octocat"}
            },
            wantStatus: http.StatusOK,
        },
        {
            name:       "store error returns 404",
            userID:     uuid.New(),
            setupStore: func(m *mockStore) {
                m.err = errors.New("not found")
            },
            wantStatus: http.StatusNotFound,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            store := &mockStore{}
            tt.setupStore(store)
            h := NewHandler(store)

            req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
            ctx := context.WithValue(req.Context(), UserIDKey, tt.userID)
            req = req.WithContext(ctx)

            w := httptest.NewRecorder()
            h.HandleMe(w, req)

            if w.Code != tt.wantStatus {
                t.Errorf("got status %d, want %d", w.Code, tt.wantStatus)
            }
        })
    }
}
```

### Store interface + mock pattern

Define an interface at the top of `handler_test.go` (or in `handler.go` if the interface is stable and worth exporting):

```go
// In handler_test.go (or handler.go)
type userGetter interface {
    GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
}

// Mock
type mockStore struct {
    user *User
    err  error
}

func (m *mockStore) GetUserByID(_ context.Context, _ uuid.UUID) (*User, error) {
    return m.user, m.err
}
```

Change the handler to accept the interface instead of `*Store` when testability matters:

```go
type Handler struct {
    store userGetter
}
```

The real `*Store` satisfies `userGetter` automatically — no registration needed.

### Handler testing with httptest

```go
func TestHandleListRepositories_empty(t *testing.T) {
    store := &mockStore{repos: []*Repository{}}
    h := NewHandler(store)

    req := httptest.NewRequest(http.MethodGet, "/api/v1/repositories", nil)
    w := httptest.NewRecorder()
    h.HandleListRepositories(w, req)

    if w.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", w.Code)
    }

    var result []Repository
    if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
        t.Fatalf("failed to decode response: %v", err)
    }
    if len(result) != 0 {
        t.Errorf("expected empty slice, got %d items", len(result))
    }
}
```

### Running tests

```bash
# all packages
go test ./...

# single package with verbose output
go test -v ./internal/auth/...

# coverage
go test -cover ./...

# coverage report in browser
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```

### What NOT to do

- Do not use `testify` unless the team explicitly adopts it — prefer stdlib `testing`.
- Do not spin up a real database in unit tests. Use the mock pattern above.
- Do not skip `t.Run` for table-driven tests — each case must have a name.
- Do not assert on exact error messages — assert on status codes and response shapes.
