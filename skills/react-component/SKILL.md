---
name: react-component
description: "How to create React components in Atlas: shadcn primitives, dark zinc theme, path aliases"
metadata:
  keywords:
    - react
    - component
    - shadcn
    - tailwind
    - typescript
    - ui
    - zinc
license: MIT
---

# React Component

## When to Use

Load this skill when:
- Creating a new UI component or page in `web/src/`
- Composing shadcn primitives into feature components
- Applying styles using the dark zinc theme
- Deciding where a component file should live

## Rules

### Component taxonomy

| Type | File path | Export |
|------|-----------|--------|
| shadcn primitive | `@/components/ui/{name}.tsx` | named |
| Feature component | `@/components/{FeatureName}.tsx` | named |
| Page component | `@/components/{PageName}Page.tsx` | default |

Pages use `export default`. Shared/feature components use named exports.

### Use shadcn, never rebuild

For buttons, cards, inputs, dialogs, dropdowns, avatars, badges, tooltips — always reach for `@/components/ui/`. Never hand-roll these.

```tsx
// Correct
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader } from "@/components/ui/card"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"

// Wrong — do not build custom button divs, hand-rolled cards, etc.
```

Use `Button asChild` to render an anchor as a button (avoids nested interactive elements):

```tsx
<Button asChild size="lg">
  <a href="/api/v1/auth/github/login">Sign in with GitHub</a>
</Button>
```

### Path aliases

Always use `@/` — never relative imports from `src/`.

```ts
// Correct
import { cn } from "@/lib/utils"
import { apiFetch } from "@/lib/auth"
import { Button } from "@/components/ui/button"

// Wrong
import { cn } from "../../lib/utils"
```

### cn() for conditional classes

```tsx
import { cn } from "@/lib/utils"

<div className={cn(
  "flex items-center gap-2",
  isActive && "text-zinc-100",
  !isActive && "text-zinc-500"
)} />
```

Never string-concatenate class names. Always use `cn()`.

### Dark zinc theme

The entire UI uses a dark zinc palette. Treat these as the baseline:

| Token | Usage |
|-------|-------|
| `bg-zinc-950` | Page/card backgrounds |
| `bg-zinc-900` | Elevated surfaces |
| `border-zinc-800` | Borders, dividers |
| `text-zinc-100` | Primary text |
| `text-zinc-400` | Secondary text |
| `text-zinc-500` | Muted / hints |

Never use `gray-*` or `slate-*`. Zinc only.

### Props interface

Define props above the component function, not inline:

```tsx
interface Props {
  user: User
  onLogout: () => void
}

export default function DashboardPage({ user, onLogout }: Props) {
```

### Component skeleton

```tsx
import { cn } from "@/lib/utils"
import { Card, CardContent } from "@/components/ui/card"

interface Props {
  title: string
  className?: string
}

export function RepositoryCard({ title, className }: Props) {
  return (
    <Card className={cn("border-zinc-800 bg-zinc-900", className)}>
      <CardContent className="p-4">
        <p className="text-zinc-100 font-medium">{title}</p>
      </CardContent>
    </Card>
  )
}
```

### Page skeleton

```tsx
import { Button } from "@/components/ui/button"

interface Props {
  onBack: () => void
}

export default function RepositoriesPage({ onBack }: Props) {
  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-100">
      <header className="border-b border-zinc-800 px-6 py-4">
        <div className="max-w-7xl mx-auto flex items-center justify-between">
          <h1 className="text-xl font-bold tracking-tight">Atlas</h1>
          <Button variant="ghost" size="sm" onClick={onBack}
            className="text-zinc-500 hover:text-zinc-300">
            Back
          </Button>
        </div>
      </header>
      <main className="max-w-7xl mx-auto px-6 py-12">
        {/* content */}
      </main>
    </div>
  )
}
```

### What NOT to do

- Do not import from `react` unless you need `useState`, `useEffect`, etc. — JSX needs no import in React 19.
- Do not use inline `style={{}}` objects — use Tailwind classes.
- Do not use `gray-*` or hardcoded hex colors.
- Do not build custom form inputs from scratch — use shadcn `Input`, `Select`, etc.
- Do not forget `className?: string` on components intended to be composable.

Canonical references: `web/src/components/LoginPage.tsx`, `web/src/components/DashboardPage.tsx`.
