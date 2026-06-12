---
name: tdd
description: >
  Test-Driven Development workflow for TypeScript/React (Vitest) and Go components.
  Trigger: ALWAYS when implementing features, fixing bugs, or refactoring.
  This is a MANDATORY workflow, not optional.
license: Apache-2.0
metadata:
  version: "1.0"
  scope: [ui, api]
  auto_invoke:
    - "Implementing feature"
    - "Fixing bug"
    - "Refactoring code"
    - "Working on task"
    - "Modifying component"
allowed-tools: Read, Edit, Write, Glob, Grep, Bash
---

## TDD Cycle (MANDATORY)

```text
+-----------------------------------------+
|  RED -> GREEN -> REFACTOR               |
|     ^                        |          |
|     +------------------------+          |
+-----------------------------------------+
```

The question is NOT "should I write tests?" but "what tests do I need?"

---

## The Three Laws of TDD

1. **No production code** until you have a failing test
2. **No more test** than necessary to fail
3. **No more code** than necessary to pass

---

## Detect Your Stack

Before starting, identify which component you're working on:

| Working in | Stack | Runner | Test pattern |
|------------|-------|--------|-------------|
| `ui/` | TypeScript / React | Vitest + RTL | `*.test.{ts,tsx}` (co-located) |
| `api/` or `backend/` | Go | `go test` | `*_test.go` (co-located) |

---

## Phase 0: Assessment (ALWAYS FIRST)

Before writing ANY code:

### UI (`ui/`)

```bash
# 1. Find existing tests
fd "*.test.tsx" src/components/feature/

# 2. Check coverage
pnpm test:coverage -- components/feature/

# 3. Read existing tests
```

### Go (`api/`)

```bash
# 1. Find existing tests
fd "_test.go" ./internal/...

# 2. Run specific test
go test ./internal/service/... -v -run TestUserService

# 3. Read existing tests
```

### Decision Tree (All Stacks)

```text
+------------------------------------------+
|     Does test file exist for this code?  |
+----------+-----------------------+-------+
           | NO                    | YES
           v                       v
+------------------+    +------------------+
| CREATE test file |    | Check coverage   |
| -> Phase 1: RED  |    | for your change  |
+------------------+    +--------+---------+
                                 |
                        +--------+--------+
                        | Missing cases?  |
                        +---+---------+---+
                            | YES     | NO
                            v         v
                    +-----------+ +-----------+
                    | ADD tests | | Proceed   |
                    | Phase 1   | | Phase 2   |
                    +-----------+ +-----------+
```

---

## Phase 1: RED - Write Failing Tests

### For NEW Functionality

#### UI (Vitest)

```typescript
describe("PriceCalculator", () => {
  it("should return 0 for quantities below threshold", () => {
    // Given
    const quantity = 3;

    // When
    const result = calculateDiscount(quantity);

    // Then
    expect(result).toBe(0);
  });
});
```

#### Go

```go
func TestCalculateDiscount_BelowThreshold(t *testing.T) {
    // Given
    quantity := 3

    // When
    result := calculateDiscount(quantity)

    // Then
    if result != 0 {
        t.Errorf("expected 0, got %d", result)
    }
}

// Table-driven style (preferred for multiple inputs)
func TestCalculateDiscount(t *testing.T) {
    tests := []struct {
        name     string
        quantity int
        want     float64
    }{
        {"below threshold", 3, 0},
        {"at threshold", 10, 0.1},
        {"above threshold", 20, 0.15},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := calculateDiscount(tt.quantity)
            if got != tt.want {
                t.Errorf("calculateDiscount(%d) = %v, want %v", tt.quantity, got, tt.want)
            }
        })
    }
}
```

**Run → MUST fail:** Test references code that doesn't exist yet.

### For BUG FIXES

Write a test that **reproduces the bug** first:

**UI:** `expect(() => render(<DatePicker value={null} />)).not.toThrow();`

**Go:** `assert.Equal(t, "FAIL", result.Status) // Currently returns "PASS" incorrectly`

Run → Should FAIL (reproducing the bug).

### For REFACTORING

Capture ALL current behavior BEFORE refactoring:

```text
# Any stack: run ALL existing tests, they should PASS
# This is your safety net - if any fail after refactoring, you broke something
```

Run → All should PASS (baseline).

---

## Phase 2: GREEN - Minimum Code

Write the MINIMUM code to make the test pass. Hardcoding is valid for the first test.

**UI:**

```typescript
// Test expects calculateDiscount(100, 10) === 10
function calculateDiscount() {
  return 10; // FAKE IT - hardcoded is valid for first test
}
```

**Go:**

```go
// Test expects calculateDiscount(3) == 0
func calculateDiscount(quantity int) float64 {
    return 0 // FAKE IT - hardcoded is valid for first test
}
```

**This passes. But we're not done...**

---

## Phase 3: Triangulation (CRITICAL)

**One test allows faking. Multiple tests FORCE real logic.**

Add tests with different inputs that break the hardcoded value:

| Scenario | Required? |
|----------|-----------|
| Happy path | YES |
| Zero/empty values | YES |
| Boundary values | YES |
| Different valid inputs | YES (breaks fake) |
| Error conditions | YES |

**UI:**

```typescript
it("should calculate 10% discount", () => {
  expect(calculateDiscount(100, 10)).toBe(10);
});

// ADD - breaks the fake:
it("should calculate 15% on 200", () => {
  expect(calculateDiscount(200, 15)).toBe(30);
});

it("should return 0 for 0% rate", () => {
  expect(calculateDiscount(100, 0)).toBe(0);
});
```

**Go (table-driven triangulation):**

```go
// Different inputs -> break hardcoded return
{"below threshold", 3, 0},       // fake passes
{"at threshold", 10, 0.1},       // breaks the fake!
{"above threshold", 20, 0.15},   // further breaks it
```

**Now fake BREAKS → Real implementation required.**

---

## Phase 4: REFACTOR

Tests GREEN → Improve code quality WITHOUT changing behavior.

- Extract functions/methods
- Improve naming
- Add types/validation
- Reduce duplication

Run tests after EACH change → Must stay GREEN.

---

## Quick Reference

```text
+------------------------------------------------+
|                 TDD WORKFLOW                    |
+------------------------------------------------+
| 0. ASSESS: What tests exist? What's missing?   |
|                                                |
| 1. RED: Write ONE failing test                 |
|    +-- Run -> Must fail with clear error       |
|                                                |
| 2. GREEN: Write MINIMUM code to pass           |
|    +-- Fake It is valid for first test         |
|                                                |
| 3. TRIANGULATE: Add tests that break the fake  |
|    +-- Different inputs, edge cases            |
|                                                |
| 4. REFACTOR: Improve with confidence           |
|    +-- Tests stay green throughout             |
|                                                |
| 5. REPEAT: Next behavior/requirement           |
+------------------------------------------------+
```

---

## Anti-Patterns (NEVER DO)

```typescript
// ANY language:

// 1. Code first, tests after
function newFeature() { ... }  // Then writing tests = USELESS

// 2. Skip triangulation
// Single test allows faking forever

// 3. Test implementation details
assert component.state.is_loading === true   // BAD - test behavior, not internals
assert mock.callCount === 3                  // BAD - brittle coupling

// 4. All tests at once before any code
// Write ONE test, make it pass, THEN write the next

// 5. Giant test methods
// Each test should verify ONE behavior
```

---

## Commands by Stack

### UI (`ui/`)

```bash
pnpm test                           # Watch mode
pnpm test:run                       # Single run (CI)
pnpm test:coverage                  # Coverage report
pnpm test ComponentName             # Filter by name
```

### Go (`api/`)

```bash
go test ./...                                    # Run all tests
go test ./internal/service/... -v                # Verbose output
go test ./internal/service/... -run TestName     # Filter by name
go test -cover ./...                             # Coverage
go test -race ./...                              # Race detector
go test -bench=. ./...                           # Benchmarks
```
