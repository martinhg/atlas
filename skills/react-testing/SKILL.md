---
name: react-testing
description: "Frontend testing conventions for Atlas: Vitest 3, RTL, userEvent, auth mocking, Radix gotchas"
metadata:
  keywords:
    - react
    - testing
    - vitest
    - rtl
    - testing-library
    - userEvent
    - mock
    - jsdom
license: MIT
---

# React Testing

## When to Use

Load this skill when:
- Writing tests for Atlas React components
- Mocking `@/lib/auth` functions
- Debugging test failures involving Radix UI components
- Setting up a new test file

## Rules

### Stack

- **Framework**: Vitest 3
- **Rendering**: `@testing-library/react`
- **Interactions**: `@testing-library/user-event`
- **Assertions**: `@testing-library/jest-dom` (auto-imported via `src/test/setup.ts`)

### File location

```
web/src/
  components/
    LoginPage.tsx
    __tests__/
      LoginPage.test.tsx    ← test lives here
  lib/
    auth.ts
    __tests__/
      auth.test.ts
```

Always `__tests__/` subdirectory, never `*.spec.tsx` next to the source file.

### Imports

```tsx
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, it, expect, vi, beforeEach } from "vitest"
```

Always import from `@testing-library/react`, not from `@testing-library/dom`.

### Query priority (RTL contract)

Use queries in this order — stop at the first one that works:

1. `getByRole` — preferred, accessible
2. `getByLabelText` — form fields
3. `getByText` — visible text content
4. `getByTestId` — last resort, add `data-testid` only when no semantic query works

```tsx
// Correct
screen.getByRole("button", { name: /sign in with github/i })
screen.getByRole("heading", { name: /atlas/i })

// Wrong — avoid unless no semantic alternative exists
screen.getByTestId("login-button")
```

### User interactions

Always use `userEvent`, never `fireEvent`:

```tsx
const user = userEvent.setup()

await user.click(screen.getByRole("button", { name: /sign out/i }))
await user.type(screen.getByLabelText("Search"), "atlas")
```

### Mocking auth functions

```tsx
import { vi, beforeEach } from "vitest"
import * as auth from "@/lib/auth"

vi.mock("@/lib/auth")

beforeEach(() => {
  vi.mocked(auth.fetchCurrentUser).mockResolvedValue({
    id: "user-1",
    github_id: 12345,
    login: "octocat",
    name: "The Octocat",
  })
  vi.mocked(auth.hasRefreshToken).mockReturnValue(true)
  vi.mocked(auth.clearAuth).mockImplementation(() => {})
})
```

`vi.mock('@/lib/auth')` auto-mocks all exports. Override only what the test needs.

### Radix UI / AvatarImage gotcha

**`AvatarImage` never fires `load` in jsdom.** The image src never resolves, so `AvatarImage` never renders — `AvatarFallback` always shows instead. Always test for the fallback, never the image:

```tsx
// Correct
expect(screen.getByText("OC")).toBeInTheDocument() // fallback initials

// Wrong — will never appear in jsdom
expect(screen.getByRole("img", { name: /octocat/i })).toBeVisible()
```

### Node 26 localStorage fix

`src/test/setup.ts` contains a fix for Node 26 where `localStorage` is undefined in jsdom. Do not replicate this fix in test files — it is already applied globally. Do not remove it.

### Async assertions

Use `waitFor` when the component has async state updates:

```tsx
await waitFor(() => {
  expect(screen.getByText("Welcome, octocat")).toBeInTheDocument()
})
```

Do not use `act` directly — RTL wraps state updates automatically. If you find yourself reaching for `act`, the test likely needs `waitFor` instead.

### Running tests

```bash
# run all tests once
pnpm test

# watch mode for development
pnpm test:watch

# coverage
pnpm test --coverage
```

### Component test skeleton

```tsx
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, it, expect, vi, beforeEach } from "vitest"
import * as auth from "@/lib/auth"
import DashboardPage from "@/components/DashboardPage"

vi.mock("@/lib/auth")

const mockUser: auth.User = {
  id: "user-1",
  github_id: 1,
  login: "octocat",
  name: "The Octocat",
}

describe("DashboardPage", () => {
  const onLogout = vi.fn()

  beforeEach(() => {
    vi.mocked(auth.clearAuth).mockImplementation(() => {})
    onLogout.mockClear()
  })

  it("renders the user login", () => {
    render(<DashboardPage user={mockUser} onLogout={onLogout} />)
    expect(screen.getByText("octocat")).toBeInTheDocument()
  })

  it("calls onLogout when sign out is clicked", async () => {
    const user = userEvent.setup()
    render(<DashboardPage user={mockUser} onLogout={onLogout} />)
    await user.click(screen.getByRole("button", { name: /sign out/i }))
    expect(onLogout).toHaveBeenCalledOnce()
  })
})
```

Canonical reference: `web/src/test/setup.ts`.
