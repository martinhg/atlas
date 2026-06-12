---
name: react-best-practices
description: React performance optimization guidelines for client-side React applications (Vite SPA). Use when writing, reviewing, or refactoring React components to ensure optimal performance patterns. Triggers on tasks involving components, data fetching, bundle optimization, or performance improvements.
license: MIT
metadata:
  author: vercel (adapted)
  version: "1.0.0"
---

# React Best Practices

Performance optimization guide for client-side React applications. Contains ~37 rules across 7 categories, prioritized by impact.

**Note:** This skill covers client-side React patterns only. Server Components, Server Actions, `React.cache()`, `after()`, and RSC-specific patterns are excluded — this is for Vite SPA architectures.

## When to Apply

Reference these guidelines when:
- Writing new React components
- Implementing client-side data fetching
- Reviewing code for performance issues
- Refactoring existing React code
- Optimizing bundle size or load times

## Rule Categories by Priority

| Priority | Category | Impact | Rules |
|----------|----------|--------|-------|
| 1 | Eliminating Waterfalls | CRITICAL | 5 |
| 2 | Bundle Size Optimization | CRITICAL | 5 |
| 3 | Client-Side Data Fetching | MEDIUM-HIGH | 4 |
| 4 | Re-render Optimization | MEDIUM | 12 |
| 5 | Rendering Performance | MEDIUM | 9 |
| 6 | JavaScript Performance | LOW-MEDIUM | 12 |
| 7 | Advanced Patterns | LOW | 3 |

---

## 1. Eliminating Waterfalls (CRITICAL)

### `async-defer-await` — Defer Await Until Needed

Move `await` operations into the branches where they're actually used to avoid blocking code paths that don't need them.

**Incorrect (blocks both branches):**

```typescript
async function handleRequest(userId: string, skipProcessing: boolean) {
  const userData = await fetchUserData(userId)
  if (skipProcessing) {
    return { skipped: true }  // Still waited for userData
  }
  return processUserData(userData)
}
```

**Correct (only blocks when needed):**

```typescript
async function handleRequest(userId: string, skipProcessing: boolean) {
  if (skipProcessing) {
    return { skipped: true }
  }
  const userData = await fetchUserData(userId)
  return processUserData(userData)
}
```

---

### `async-parallel` — Promise.all() for Independent Operations

When async operations have no interdependencies, execute them concurrently using `Promise.all()`. **Impact: 2–10× improvement.**

**Incorrect (sequential — 3 round trips):**

```typescript
const user = await fetchUser()
const posts = await fetchPosts()
const comments = await fetchComments()
```

**Correct (parallel — 1 round trip):**

```typescript
const [user, posts, comments] = await Promise.all([
  fetchUser(),
  fetchPosts(),
  fetchComments()
])
```

---

### `async-dependencies` — Dependency-Based Parallelization

For operations with partial dependencies, use `better-all` to maximize parallelism.

**Incorrect (profile waits for config unnecessarily):**

```typescript
const [user, config] = await Promise.all([fetchUser(), fetchConfig()])
const profile = await fetchProfile(user.id)
```

**Correct (config and profile run in parallel):**

```typescript
import { all } from 'better-all'

const { user, config, profile } = await all({
  async user() { return fetchUser() },
  async config() { return fetchConfig() },
  async profile() { return fetchProfile((await this.$.user).id) }
})
```

Reference: https://github.com/shuding/better-all

---

### `async-api-routes` — Start Promises Early, Await Late

In API handlers, start independent operations immediately, even if you don't await them yet.

**Incorrect (config waits for auth, data waits for both):**

```typescript
export async function GET(request: Request) {
  const session = await auth()
  const config = await fetchConfig()
  const data = await fetchData(session.user.id)
  return Response.json({ data, config })
}
```

**Correct (auth and config start immediately):**

```typescript
export async function GET(request: Request) {
  const sessionPromise = auth()
  const configPromise = fetchConfig()
  const session = await sessionPromise
  const [config, data] = await Promise.all([
    configPromise,
    fetchData(session.user.id)
  ])
  return Response.json({ data, config })
}
```

---

### `async-suspense-boundaries` — Strategic Suspense Boundaries

Instead of awaiting all data before returning JSX, use Suspense to show layout immediately while data loads.

```tsx
function Page() {
  return (
    <div>
      <Sidebar />
      <Header />
      <Suspense fallback={<Skeleton />}>
        <DataDisplay />  {/* Only this waits for data */}
      </Suspense>
      <Footer />
    </div>
  )
}
```

Sidebar, Header, and Footer render immediately. Use `use(promise)` to share a promise across multiple Suspense children without re-fetching.

---

## 2. Bundle Size Optimization (CRITICAL)

### `bundle-barrel-imports` — Avoid Barrel File Imports

Import directly from source files instead of barrel files. Barrel files can have **up to 10,000 re-exports**, costing 200–800ms on every cold start.

**Incorrect (imports entire library):**

```tsx
import { Check, X, Menu } from 'lucide-react'
// Loads 1,583 modules — ~2.8s extra in dev
```

**Correct (imports only what you need):**

```tsx
import Check from 'lucide-react/dist/esm/icons/check'
import X from 'lucide-react/dist/esm/icons/x'
import Menu from 'lucide-react/dist/esm/icons/menu'
```

Commonly affected: `lucide-react`, `@mui/material`, `@mui/icons-material`, `@tabler/icons-react`, `react-icons`, `@radix-ui/react-*`, `lodash`, `date-fns`.

---

### `bundle-dynamic-imports` — Dynamic Imports for Heavy Components

Use `React.lazy()` (or your router's lazy) to lazy-load large components not needed on initial render.

```tsx
import { lazy, Suspense } from 'react'

const MonacoEditor = lazy(() =>
  import('./monaco-editor').then(m => ({ default: m.MonacoEditor }))
)

function CodePanel({ code }: { code: string }) {
  return (
    <Suspense fallback={<Skeleton />}>
      <MonacoEditor value={code} />
    </Suspense>
  )
}
```

---

### `bundle-defer-third-party` — Defer Non-Critical Third-Party Libraries

Analytics, logging, and error tracking don't block user interaction. Load them after hydration.

```tsx
import { useEffect } from 'react'

function App() {
  useEffect(() => {
    // Load analytics after hydration — doesn't block initial render
    import('@vercel/analytics/react').then(({ inject }) => inject())
  }, [])
  // ...
}
```

---

### `bundle-conditional` — Conditional Module Loading

Load large data or modules only when a feature is activated.

```tsx
function AnimationPlayer({ enabled, setEnabled }: Props) {
  const [frames, setFrames] = useState<Frame[] | null>(null)

  useEffect(() => {
    if (enabled && !frames) {
      import('./animation-frames.js')
        .then(mod => setFrames(mod.frames))
        .catch(() => setEnabled(false))
    }
  }, [enabled, frames, setEnabled])

  if (!frames) return <Skeleton />
  return <Canvas frames={frames} />
}
```

---

### `bundle-preload` — Preload Based on User Intent

Preload heavy bundles before they're needed to reduce perceived latency.

```tsx
function EditorButton({ onClick }: { onClick: () => void }) {
  const preload = () => {
    void import('./monaco-editor')
  }

  return (
    <button onMouseEnter={preload} onFocus={preload} onClick={onClick}>
      Open Editor
    </button>
  )
}
```

---

## 3. Client-Side Data Fetching (MEDIUM-HIGH)

### `client-swr-dedup` — Use SWR/TanStack Query for Automatic Deduplication

SWR and TanStack Query enable request deduplication, caching, and revalidation across component instances.

**Incorrect (no deduplication — each instance fetches):**

```tsx
function UserList() {
  const [users, setUsers] = useState([])
  useEffect(() => {
    fetch('/api/users').then(r => r.json()).then(setUsers)
  }, [])
}
```

**Correct (multiple instances share one request):**

```tsx
import useSWR from 'swr'

function UserList() {
  const { data: users } = useSWR('/api/users', fetcher)
}
```

---

### `client-event-listeners` — Deduplicate Global Event Listeners

Use `useSWRSubscription()` to share global event listeners across component instances instead of registering one per mount.

```tsx
import useSWRSubscription from 'swr/subscription'
const keyCallbacks = new Map<string, Set<() => void>>()

function useKeyboardShortcut(key: string, callback: () => void) {
  useEffect(() => {
    if (!keyCallbacks.has(key)) keyCallbacks.set(key, new Set())
    keyCallbacks.get(key)!.add(callback)
    return () => {
      const set = keyCallbacks.get(key)
      if (set) { set.delete(callback); if (!set.size) keyCallbacks.delete(key) }
    }
  }, [key, callback])

  useSWRSubscription('global-keydown', () => {
    const handler = (e: KeyboardEvent) => {
      if (e.metaKey && keyCallbacks.has(e.key)) {
        keyCallbacks.get(e.key)!.forEach(cb => cb())
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  })
}
```

---

### `client-localstorage-schema` — Version and Minimize localStorage Data

Add version prefix to keys and store only needed fields. Prevents schema conflicts and accidental storage of sensitive data.

```typescript
const VERSION = 'v2'

function saveConfig(config: { theme: string; language: string }) {
  try {
    localStorage.setItem(`userConfig:${VERSION}`, JSON.stringify(config))
  } catch {
    // Throws in incognito/private browsing, quota exceeded, or disabled
  }
}

function loadConfig() {
  try {
    const data = localStorage.getItem(`userConfig:${VERSION}`)
    return data ? JSON.parse(data) : null
  } catch {
    return null
  }
}
```

Always wrap `getItem()`/`setItem()` in try-catch — they throw in incognito mode and when quota is exceeded.

---

### `client-passive-event-listeners` — Passive Event Listeners for Scrolling

Add `{ passive: true }` to touch and wheel event listeners. Browsers normally wait for listeners to finish before scrolling (to check for `preventDefault`), causing delay.

```typescript
useEffect(() => {
  const handleTouch = (e: TouchEvent) => console.log(e.touches[0].clientX)
  const handleWheel = (e: WheelEvent) => console.log(e.deltaY)

  document.addEventListener('touchstart', handleTouch, { passive: true })
  document.addEventListener('wheel', handleWheel, { passive: true })

  return () => {
    document.removeEventListener('touchstart', handleTouch)
    document.removeEventListener('wheel', handleWheel)
  }
}, [])
```

Use passive only when you don't call `preventDefault()`.

---

## 4. Re-render Optimization (MEDIUM)

### `rerender-defer-reads` — Defer State Reads to Usage Point

Don't subscribe to dynamic state if you only read it inside callbacks.

```tsx
// Incorrect: subscribes to all searchParams changes
function ShareButton({ chatId }: { chatId: string }) {
  const searchParams = useSearchParams()
  const handleShare = () => shareChat(chatId, { ref: searchParams.get('ref') })
  return <button onClick={handleShare}>Share</button>
}

// Correct: reads on demand, no subscription
function ShareButton({ chatId }: { chatId: string }) {
  const handleShare = () => {
    const ref = new URLSearchParams(window.location.search).get('ref')
    shareChat(chatId, { ref })
  }
  return <button onClick={handleShare}>Share</button>
}
```

---

### `rerender-memo` — Extract to Memoized Components

Extract expensive work into memoized components to enable early returns before computation.

```tsx
// Incorrect: computes avatar even when loading
function Profile({ user, loading }: Props) {
  const avatar = useMemo(() => {
    const id = computeAvatarId(user)
    return <Avatar id={id} />
  }, [user])
  if (loading) return <Skeleton />
  return <div>{avatar}</div>
}

// Correct: skips computation when loading
const UserAvatar = memo(function UserAvatar({ user }: { user: User }) {
  const id = useMemo(() => computeAvatarId(user), [user])
  return <Avatar id={id} />
})

function Profile({ user, loading }: Props) {
  if (loading) return <Skeleton />
  return <div><UserAvatar user={user} /></div>
}
```

**Note:** If React Compiler is enabled, manual `memo()`/`useMemo()` is not necessary.

---

### `rerender-dependencies` — Narrow Effect Dependencies

Specify primitive dependencies instead of objects to minimize effect re-runs.

```tsx
// Incorrect: re-runs on any user field change
useEffect(() => { console.log(user.id) }, [user])

// Correct: re-runs only when id changes
useEffect(() => { console.log(user.id) }, [user.id])
```

---

### `rerender-derived-state` — Subscribe to Derived State

Subscribe to derived boolean state instead of continuous values to reduce re-render frequency.

```tsx
// Incorrect: re-renders on every pixel change
function Sidebar() {
  const width = useWindowWidth()  // updates on every pixel
  const isMobile = width < 768
  return <nav className={isMobile ? 'mobile' : 'desktop'} />
}

// Correct: re-renders only when boolean changes
function Sidebar() {
  const isMobile = useMediaQuery('(max-width: 767px)')
  return <nav className={isMobile ? 'mobile' : 'desktop'} />
}
```

---

### `rerender-functional-setstate` — Use Functional setState Updates

Use the functional update form when updating state based on current state. Prevents stale closures and unnecessary callback recreations.

```tsx
// Incorrect: requires state as dependency, recreated on every items change
const addItems = useCallback((newItems: Item[]) => {
  setItems([...items, ...newItems])
}, [items])

// Correct: stable callback, always uses latest state
const addItems = useCallback((newItems: Item[]) => {
  setItems(curr => [...curr, ...newItems])
}, [])
```

---

### `rerender-lazy-state-init` — Use Lazy State Initialization

Pass a function to `useState` for expensive initial values — the initializer runs on every render without the function form.

```tsx
// Incorrect: buildSearchIndex() runs on EVERY render
const [searchIndex, setSearchIndex] = useState(buildSearchIndex(items))

// Correct: runs only once on initial render
const [searchIndex, setSearchIndex] = useState(() => buildSearchIndex(items))
```

---

### `rerender-transitions` — Use Transitions for Non-Urgent Updates

Mark frequent, non-urgent state updates as transitions to maintain UI responsiveness.

```tsx
import { startTransition } from 'react'

function ScrollTracker() {
  const [scrollY, setScrollY] = useState(0)
  useEffect(() => {
    const handler = () => {
      startTransition(() => setScrollY(window.scrollY))
    }
    window.addEventListener('scroll', handler, { passive: true })
    return () => window.removeEventListener('scroll', handler)
  }, [])
}
```

---

### `rerender-memo-with-default-value` — Hoist Default Non-Primitive Props

When a memoized component has a default value for a non-primitive optional parameter, omitting it breaks memoization (new reference on every render).

```tsx
// Incorrect: inline function creates new reference every render
const UserAvatar = memo(function UserAvatar({ onClick = () => {} }: { onClick?: () => void }) {
  // ...
})

// Correct: module-level constant = stable reference
const NOOP = () => {}
const UserAvatar = memo(function UserAvatar({ onClick = NOOP }: { onClick?: () => void }) {
  // ...
})
```

---

### `rerender-derived-state-no-effect` — Derive State During Render

Compute values from current props/state during rendering rather than storing them as state or updating them via effects.

```tsx
// Incorrect: storing derived state + syncing with useEffect
const [fullName, setFullName] = useState('')
useEffect(() => { setFullName(`${firstName} ${lastName}`) }, [firstName, lastName])

// Correct: compute during render
const fullName = `${firstName} ${lastName}`
```

---

### `rerender-simple-expression-in-memo` — Avoid useMemo for Simple Primitives

Don't wrap a simple expression with a primitive result type in `useMemo`. The hook overhead exceeds any benefit.

```tsx
// Incorrect: useMemo for a simple boolean
const isLoading = useMemo(() => user.isLoading || notifications.isLoading, [user, notifications])

// Correct: compute directly
const isLoading = user.isLoading || notifications.isLoading
```

---

### `rerender-move-effect-to-event` — Put Interaction Logic in Event Handlers

Place side effects triggered by user actions directly in event handlers rather than modeling them as state with `useEffect`.

```tsx
// Incorrect: state mediating a side effect
const [submitted, setSubmitted] = useState(false)
useEffect(() => { if (submitted) sendAnalytics() }, [submitted])
const handleSubmit = () => { setSubmitted(true) }

// Correct: direct in handler
const handleSubmit = () => { sendAnalytics() }
```

---

### `rerender-use-ref-transient-values` — Use useRef for Transient Values

Use `useRef` instead of `useState` for values that change frequently without requiring UI updates (mouse position, timers, temporary flags).

```tsx
// Incorrect: causes re-render on every mouse move
const [pos, setPos] = useState({ x: 0, y: 0 })
const handleMove = (e: MouseEvent) => setPos({ x: e.clientX, y: e.clientY })

// Correct: ref update doesn't trigger re-render
const posRef = useRef({ x: 0, y: 0 })
const cursorRef = useRef<HTMLDivElement>(null)
const handleMove = (e: MouseEvent) => {
  posRef.current = { x: e.clientX, y: e.clientY }
  if (cursorRef.current) {
    cursorRef.current.style.transform = `translate(${e.clientX}px, ${e.clientY}px)`
  }
}
```

---

## 5. Rendering Performance (MEDIUM)

### `rendering-animate-svg-wrapper` — Animate SVG Wrapper

Many browsers don't hardware-accelerate CSS3 animations on SVG elements. Wrap SVG in a `<div>` and animate the wrapper.

```tsx
// Correct (hardware-accelerated wrapper)
function LoadingSpinner() {
  return (
    <div className="animate-spin">
      <svg width="24" height="24" viewBox="0 0 24 24">
        <circle cx="12" cy="12" r="10" stroke="currentColor" />
      </svg>
    </div>
  )
}
```

---

### `rendering-content-visibility` — CSS content-visibility for Long Lists

Apply `content-visibility: auto` to defer off-screen rendering. For 1000 items, browsers skip layout/paint for ~990 off-screen items (10× faster initial render).

```css
.message-item {
  content-visibility: auto;
  contain-intrinsic-size: 0 80px;
}
```

---

### `rendering-hoist-jsx` — Hoist Static JSX Elements

Extract static JSX outside components to avoid re-creation on every render.

```tsx
// Correct: reuses same element
const loadingSkeleton = <div className="animate-pulse h-20 bg-gray-200" />

function Container() {
  return <div>{loading && loadingSkeleton}</div>
}
```

**Note:** React Compiler automates this.

---

### `rendering-svg-precision` — Reduce SVG Precision

Reduce SVG coordinate precision to decrease file size.

```bash
npx svgo --precision=1 --multipass icon.svg
```

---

### `rendering-hydration-no-flicker` — Prevent Hydration Mismatch Without Flickering

For client-only data (localStorage, cookies), inject a synchronous script to update the DOM before React hydrates.

```tsx
function ThemeWrapper({ children }: { children: ReactNode }) {
  return (
    <>
      <div id="theme-wrapper">{children}</div>
      <script
        dangerouslySetInnerHTML={{
          __html: `(function(){try{var t=localStorage.getItem('theme')||'light';var el=document.getElementById('theme-wrapper');if(el)el.className=t;}catch(e){}})();`,
        }}
      />
    </>
  )
}
```

---

### `rendering-activity` — Use Activity Component for Show/Hide

Use React's `<Activity>` to preserve state/DOM for expensive components that frequently toggle visibility.

```tsx
import { Activity } from 'react'

function Dropdown({ isOpen }: Props) {
  return (
    <Activity mode={isOpen ? 'visible' : 'hidden'}>
      <ExpensiveMenu />
    </Activity>
  )
}
```

---

### `rendering-conditional-render` — Ternary Over && for Conditionals

Use explicit ternary operators instead of `&&` when the condition can be `0`, `NaN`, or other falsy values that render as text.

```tsx
// Incorrect: renders "0" when count is 0
{count && <span className="badge">{count}</span>}

// Correct: renders nothing when count is 0
{count > 0 ? <span className="badge">{count}</span> : null}
```

---

### `rendering-hydration-suppress-warning` — Suppress Expected Hydration Mismatches

Use `suppressHydrationWarning` only for known, intentional server/client differences (timestamps, random IDs). Do not use to hide real bugs.

```tsx
function Timestamp() {
  return <span suppressHydrationWarning>{new Date().toLocaleString()}</span>
}
```

---

### `rendering-usetransition-loading` — useTransition Over Manual Loading States

Replace manual `isLoading` state with `useTransition` for cleaner code. Provides automatic pending state, error resilience, and transition cancellation.

```tsx
// Correct: built-in isPending, automatic management
const [isPending, startTransition] = useTransition()

const handleSearch = (query: string) => {
  startTransition(() => {
    setResults(computeResults(query))
  })
}
```

---

## 6. JavaScript Performance (LOW-MEDIUM)

### `js-batch-dom-css` — Batch DOM CSS Changes

Avoid changing styles one property at a time. Group multiple CSS changes together via classes or `cssText` to minimize browser reflows.

```tsx
// Correct: toggle class = single reflow
function Box({ isHighlighted }: { isHighlighted: boolean }) {
  return <div className={isHighlighted ? 'highlighted-box' : ''}>Content</div>
}
```

---

### `js-index-maps` — Build Index Maps for Repeated Lookups

Multiple `.find()` calls by the same key should use a Map. For 1000 orders × 1000 users: 1M ops → 2K ops.

```typescript
function processOrders(orders: Order[], users: User[]) {
  const userById = new Map(users.map(u => [u.id, u]))
  return orders.map(order => ({ ...order, user: userById.get(order.userId) }))
}
```

---

### `js-cache-property-access` — Cache Property Access in Loops

Cache object property lookups in hot paths.

```typescript
// Correct: 1 lookup total
const value = obj.config.settings.value
const len = arr.length
for (let i = 0; i < len; i++) {
  process(value)
}
```

---

### `js-cache-function-results` — Cache Repeated Function Calls

Use a module-level Map to cache function results when the same function is called repeatedly with the same inputs.

```typescript
const slugifyCache = new Map<string, string>()

function cachedSlugify(text: string): string {
  if (slugifyCache.has(text)) return slugifyCache.get(text)!
  const result = slugify(text)
  slugifyCache.set(text, result)
  return result
}
```

---

### `js-cache-storage` — Cache Storage API Calls

`localStorage`, `sessionStorage`, and `document.cookie` are synchronous and expensive. Cache reads in memory.

```typescript
const storageCache = new Map<string, string | null>()

function getLocalStorage(key: string) {
  if (!storageCache.has(key)) storageCache.set(key, localStorage.getItem(key))
  return storageCache.get(key)
}

function setLocalStorage(key: string, value: string) {
  localStorage.setItem(key, value)
  storageCache.set(key, value)
}
```

---

### `js-combine-iterations` — Combine Multiple Array Iterations

Multiple `.filter()` or `.map()` calls iterate the array multiple times. Combine into one loop.

```typescript
// Correct: 1 iteration instead of 3
const admins: User[] = [], testers: User[] = [], inactive: User[] = []
for (const user of users) {
  if (user.isAdmin) admins.push(user)
  if (user.isTester) testers.push(user)
  if (!user.isActive) inactive.push(user)
}
```

---

### `js-length-check-first` — Early Length Check for Array Comparisons

When comparing arrays with expensive operations, check lengths first. Different lengths = can't be equal.

```typescript
function hasChanges(current: string[], original: string[]) {
  if (current.length !== original.length) return true
  const currentSorted = current.toSorted()
  const originalSorted = original.toSorted()
  for (let i = 0; i < currentSorted.length; i++) {
    if (currentSorted[i] !== originalSorted[i]) return true
  }
  return false
}
```

---

### `js-early-exit` — Early Return from Functions

Return early when the result is determined to skip unnecessary processing.

```typescript
function validateUsers(users: User[]) {
  for (const user of users) {
    if (!user.email) return { valid: false, error: 'Email required' }
    if (!user.name) return { valid: false, error: 'Name required' }
  }
  return { valid: true }
}
```

---

### `js-hoist-regexp` — Hoist RegExp Creation

Don't create RegExp inside render. Hoist to module scope or memoize with `useMemo()`.

```tsx
// For dynamic regex: memoize
function Highlighter({ text, query }: Props) {
  const regex = useMemo(
    () => new RegExp(`(${escapeRegex(query)})`, 'gi'),
    [query]
  )
  const parts = text.split(regex)
  return <>{parts.map((part, i) => ...)}</>
}
```

**Warning:** Global regex (`/g`) has mutable `lastIndex` state — be careful when reusing.

---

### `js-min-max-loop` — Use Loop for Min/Max Instead of Sort

Finding the smallest or largest element requires only a single pass (O(n)), not a sort (O(n log n)).

```typescript
function getLatestProject(projects: Project[]) {
  if (!projects.length) return null
  let latest = projects[0]
  for (let i = 1; i < projects.length; i++) {
    if (projects[i].updatedAt > latest.updatedAt) latest = projects[i]
  }
  return latest
}
```

---

### `js-set-map-lookups` — Use Set/Map for O(1) Lookups

Convert arrays to Set/Map for repeated membership checks.

```typescript
// Incorrect: O(n) per check
const allowedIds = ['a', 'b', 'c']
items.filter(item => allowedIds.includes(item.id))

// Correct: O(1) per check
const allowedIds = new Set(['a', 'b', 'c'])
items.filter(item => allowedIds.has(item.id))
```

---

### `js-tosorted-immutable` — Use toSorted() for Immutability

`.sort()` mutates the array in place, which causes bugs with React state. Use `.toSorted()`.

```typescript
// Incorrect: mutates the users prop
const sorted = useMemo(() => users.sort((a, b) => a.name.localeCompare(b.name)), [users])

// Correct: creates new sorted array
const sorted = useMemo(() => users.toSorted((a, b) => a.name.localeCompare(b.name)), [users])
```

Available in Chrome 110+, Safari 16+, Firefox 115+, Node.js 20+. Fallback: `[...items].sort(...)`.

---

## 7. Advanced Patterns (LOW)

### `advanced-event-handler-refs` — Store Event Handlers in Refs

Store callbacks in refs when used in effects that shouldn't re-subscribe on callback changes.

```tsx
function useWindowEvent(event: string, handler: () => void) {
  const handlerRef = useRef(handler)
  useEffect(() => { handlerRef.current = handler }, [handler])

  useEffect(() => {
    const listener = () => handlerRef.current()
    window.addEventListener(event, listener)
    return () => window.removeEventListener(event, listener)
  }, [event])
}
```

Or use `useEffectEvent` on latest React for a cleaner API.

---

### `advanced-use-latest` — useLatest for Stable Callback Refs

Access latest values in callbacks without adding them to dependency arrays.

```typescript
function useLatest<T>(value: T) {
  const ref = useRef(value)
  useEffect(() => { ref.current = value }, [value])
  return ref
}

function SearchInput({ onSearch }: { onSearch: (q: string) => void }) {
  const [query, setQuery] = useState('')
  const onSearchRef = useLatest(onSearch)

  useEffect(() => {
    const timeout = setTimeout(() => onSearchRef.current(query), 300)
    return () => clearTimeout(timeout)
  }, [query])  // No onSearch in deps — stable effect, fresh callback
}
```

---

### `advanced-init-once` — Initialize App Once Per App Load

Avoid placing app-wide initialization inside `useEffect([])` — components can remount and re-run effects. Use a module-level guard.

```tsx
let didInit = false

function App() {
  useEffect(() => {
    if (didInit) return
    didInit = true
    loadFromStorage()
    checkAuthToken()
  }, [])
}
```
