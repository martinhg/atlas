---
name: zustand-5
description: >
  Zustand 5 state management patterns for Vite SPA (React + TypeScript).
  Trigger: When implementing client-side state with Zustand (stores, selectors, persist middleware, slices, devtools).
license: Apache-2.0
metadata:
  version: "1.0"
  auto_invoke: "Using Zustand stores"
allowed-tools: Read, Edit, Write, Glob, Grep, Bash
---

## Quick Start (3 Minutes)

### Install

```bash
pnpm add zustand
```

**Latest version:** zustand@5.0.8  
**Requirements:** React 18+, TypeScript 5+

### Why Zustand?

- Minimal API: only 1 function to learn (`create`)
- No boilerplate: no providers, reducers, or actions
- TypeScript-first: excellent type inference
- Fast: fine-grained subscriptions prevent unnecessary re-renders
- Flexible: middleware for persistence, devtools, and more

---

## CRITICAL: TypeScript Double Parentheses

```typescript
// WRONG: create<T>(...)
const useStore = create<MyStore>((set) => ({ ... }))

// CORRECT: create<T>()(...) — required for middleware compatibility
const useStore = create<MyStore>()((set) => ({ ... }))
```

This is the most common TypeScript gotcha. Without the double parentheses, middleware types break.

---

## Basic Store

```typescript
import { create } from "zustand";

interface CounterStore {
  count: number;
  increment: () => void;
  decrement: () => void;
  reset: () => void;
}

const useCounterStore = create<CounterStore>()((set) => ({
  count: 0,
  increment: () => set((state) => ({ count: state.count + 1 })),
  decrement: () => set((state) => ({ count: state.count - 1 })),
  reset: () => set({ count: 0 }),
}));

// Usage
function Counter() {
  const { count, increment, decrement } = useCounterStore();
  return (
    <div>
      <span>{count}</span>
      <button onClick={increment}>+</button>
      <button onClick={decrement}>-</button>
    </div>
  );
}
```

---

## Selectors (Zustand 5) — Prevent Unnecessary Re-renders

```typescript
// Select specific fields to prevent re-renders on unrelated changes
function UserName() {
  const name = useUserStore((state) => state.name);
  return <span>{name}</span>;
}

// For multiple fields, use useShallow
import { useShallow } from "zustand/react/shallow";

function UserInfo() {
  const { name, email } = useUserStore(
    useShallow((state) => ({ name: state.name, email: state.email }))
  );
  return <div>{name} - {email}</div>;
}

// AVOID: Selecting entire store (causes re-render on ANY state change)
const store = useUserStore();  // BAD
```

---

## Async Actions

```typescript
interface UserStore {
  user: User | null;
  loading: boolean;
  error: string | null;
  fetchUser: (id: string) => Promise<void>;
}

const useUserStore = create<UserStore>()((set) => ({
  user: null,
  loading: false,
  error: null,

  fetchUser: async (id) => {
    set({ loading: true, error: null });
    try {
      const response = await fetch(`/api/users/${id}`);
      const user = await response.json();
      set({ user, loading: false });
    } catch {
      set({ error: "Failed to fetch user", loading: false });
    }
  },
}));
```

---

## Persist Middleware

```typescript
import { create } from "zustand";
import { persist, createJSONStorage } from "zustand/middleware";

interface SettingsStore {
  theme: "light" | "dark";
  language: string;
  setTheme: (theme: "light" | "dark") => void;
  setLanguage: (language: string) => void;
}

const useSettingsStore = create<SettingsStore>()(
  persist(
    (set) => ({
      theme: "light",
      language: "en",
      setTheme: (theme) => set({ theme }),
      setLanguage: (language) => set({ language }),
    }),
    {
      name: "settings-storage",  // Unique localStorage key — never reuse across stores!
      storage: createJSONStorage(() => localStorage),  // defaults to localStorage
    }
  )
);
```

**Persist middleware notes:**
- Use unique names per store to avoid data collisions
- State is automatically restored on page reload
- Works with `sessionStorage` too — swap `localStorage` for `sessionStorage`
- Wrap any direct localStorage access in try-catch (throws in incognito/private browsing)

---

## Immer Middleware

```typescript
import { create } from "zustand";
import { immer } from "zustand/middleware/immer";

interface TodoStore {
  todos: Todo[];
  addTodo: (text: string) => void;
  toggleTodo: (id: string) => void;
}

const useTodoStore = create<TodoStore>()(
  immer((set) => ({
    todos: [],

    addTodo: (text) => set((state) => {
      // Mutate directly with Immer — no need to spread
      state.todos.push({ id: crypto.randomUUID(), text, done: false });
    }),

    toggleTodo: (id) => set((state) => {
      const todo = state.todos.find(t => t.id === id);
      if (todo) todo.done = !todo.done;
    }),
  }))
);
```

---

## DevTools

```typescript
import { create } from "zustand";
import { devtools } from "zustand/middleware";

const useStore = create<Store>()(
  devtools(
    (set) => ({
      // store definition
    }),
    { name: "MyStore" }  // Name shown in Redux DevTools
  )
);
```

**Middleware order:** `devtools(persist(...))` shows persist actions in DevTools.

---

## Slices Pattern

For large stores, split into slices and compose:

```typescript
// userSlice.ts
import { StateCreator } from "zustand";

interface UserSlice {
  user: User | null;
  setUser: (user: User) => void;
  clearUser: () => void;
}

const createUserSlice: StateCreator<
  UserSlice & CartSlice,  // Combined store type
  [],
  [],
  UserSlice
> = (set) => ({
  user: null,
  setUser: (user) => set({ user }),
  clearUser: () => set({ user: null }),
});

// cartSlice.ts
interface CartSlice {
  items: CartItem[];
  addItem: (item: CartItem) => void;
  removeItem: (id: string) => void;
}

const createCartSlice: StateCreator<
  UserSlice & CartSlice,
  [],
  [],
  CartSlice
> = (set) => ({
  items: [],
  addItem: (item) => set((state) => ({ items: [...state.items, item] })),
  removeItem: (id) => set((state) => ({
    items: state.items.filter(i => i.id !== id)
  })),
});

// store.ts
type Store = UserSlice & CartSlice;

const useStore = create<Store>()((...args) => ({
  ...createUserSlice(...args),
  ...createCartSlice(...args),
}));
```

**Note on slices types:** explicit `StateCreator<Combined, [], [], Slice>` is required — TypeScript inference fails without it (Known Issue #5).

---

## Outside React

```typescript
// Access store state outside components
const { count, increment } = useCounterStore.getState();
increment();

// Subscribe to changes
const unsubscribe = useCounterStore.subscribe(
  (state) => console.log("Count changed:", state.count)
);

// Later: cleanup
unsubscribe();
```

---

## Known Issues Prevention

| Issue | Symptom | Fix |
|-------|---------|-----|
| **Hydration mismatch** | "Text content does not match" | Use `_hasHydrated` flag + `onRehydrateStorage` |
| **TypeScript inference** | Types break with middleware | Use `create<T>()()` double parentheses |
| **Import error** | "createJSONStorage not exported" | Upgrade to zustand@5.0.8+ |
| **Infinite loop** | Browser freezes | Use `useShallow` or separate selectors |
| **Slices types** | StateCreator types fail | Use explicit `StateCreator<Combined, [], [], Slice>` |

---

## Common Patterns

### Reset Store

```typescript
const initialState = { user: null, loading: false, error: null }

const useUserStore = create<UserStore>()((set) => ({
  ...initialState,
  // actions...
  reset: () => set(initialState),
}));

// On logout:
useUserStore.getState().reset();
```

### Computed/Derived Values

```typescript
// Compute in selector — don't store derived state
function CartSummary() {
  const total = useCartStore((state) =>
    state.items.reduce((sum, item) => sum + item.price * item.quantity, 0)
  );
  return <div>Total: ${total}</div>;
}
```

### Parameterized Selector

```typescript
function useTodo(id: string) {
  return useTodoStore((state) => state.todos.find(t => t.id === id));
}
```

---

## Critical Rules

### Always Do

- Use `create<T>()()` (double parentheses) in TypeScript for middleware compatibility
- Define separate interfaces for state and actions
- Use selector functions to extract specific state slices
- Use `set` with updater functions for derived state: `set((state) => ({ count: state.count + 1 }))`
- Use unique names for persist middleware storage keys
- Use `useShallow` for selecting multiple values
- Keep actions pure (no side effects except state updates)
- Export only the hook — never export the store instance directly

### Never Do

- Use `create<T>(...)` (single parentheses) in TypeScript — breaks middleware types
- Mutate state directly: `set((state) => { state.count++; return state })` — use immutable updates
- Create new objects in selectors: `useStore((state) => ({ a: state.a }))` — causes infinite renders
- Use the same storage name for multiple stores — causes data collisions
- Use Zustand for server state — use TanStack Query instead

---

## Further Reading

- **Official Docs:** https://zustand.docs.pmnd.rs/
- **GitHub:** https://github.com/pmndrs/zustand
