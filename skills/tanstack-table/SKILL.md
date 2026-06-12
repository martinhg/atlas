---
name: tanstack-table
description: TanStack Table v8 headless data tables with server-side features. Use for pagination, filtering, sorting, virtualization, or encountering state management, TanStack Query coordination, URL sync errors.
license: MIT
allowed-tools: [Bash, Read, Write, Edit]
metadata:
  version: 1.0.0
  keywords:
    - tanstack table
    - react table
    - data table
    - datagrid
    - headless table
    - server-side pagination
    - server-side filtering
    - server-side sorting
    - tanstack query integration
    - virtualization
    - tanstack virtual
    - large datasets
    - table state management
    - url state sync
    - column configuration
    - typescript table
    - react table v8
    - headless ui
    - pagination
    - filtering
    - sorting
---

# TanStack Table Skill

Build production-ready, headless data tables with TanStack Table v8, optimized for server-side patterns with REST API backends.

---

## When to Use This Skill

**Auto-triggers when you mention:**
- "data table" or "datagrid"
- "server-side pagination" or "server-side filtering"
- "TanStack Table" or "React Table"
- "table with large dataset"
- "paginate/filter/sort with API"
- "virtualize table" or "large list performance"

**Use this skill when:**
- Building data tables with pagination, filtering, or sorting
- Implementing server-side table features (API-driven)
- Integrating tables with TanStack Query for data fetching
- Working with large datasets (1000+ rows) needing virtualization
- Need headless table logic without opinionated UI

---

## Quick Start

### Installation

```bash
# Core table library
pnpm add @tanstack/react-table@latest

# Optional: For virtualization (1000+ rows)
pnpm add @tanstack/react-virtual@latest

# Optional: For fuzzy/global search
pnpm add @tanstack/match-sorter-utils@latest
```

**Latest verified versions:**
- `@tanstack/react-table`: v8.21.3 (stable)
- `@tanstack/react-virtual`: v3.13.12
- `@tanstack/match-sorter-utils`: v8.21.3

**React support:** Works on React 16.8+ through React 19; React Compiler is not supported.

### Basic Client-Side Table

```typescript
import { useReactTable, getCoreRowModel, ColumnDef } from '@tanstack/react-table'
import { useMemo } from 'react'

interface User {
  id: string
  name: string
  email: string
}

const columns: ColumnDef<User>[] = [
  { accessorKey: 'id', header: 'ID' },
  { accessorKey: 'name', header: 'Name' },
  { accessorKey: 'email', header: 'Email' },
]

function UsersTable() {
  // CRITICAL: Memoize data and columns to prevent infinite re-renders
  const data = useMemo<User[]>(() => [
    { id: '1', name: 'Alice', email: 'alice@example.com' },
    { id: '2', name: 'Bob', email: 'bob@example.com' },
  ], [])

  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(), // Required
  })

  return (
    <table>
      <thead>
        {table.getHeaderGroups().map(headerGroup => (
          <tr key={headerGroup.id}>
            {headerGroup.headers.map(header => (
              <th key={header.id}>
                {header.isPlaceholder ? null : header.column.columnDef.header}
              </th>
            ))}
          </tr>
        ))}
      </thead>
      <tbody>
        {table.getRowModel().rows.map(row => (
          <tr key={row.id}>
            {row.getVisibleCells().map(cell => (
              <td key={cell.id}>
                {cell.renderValue()}
              </td>
            ))}
          </tr>
        ))}
      </tbody>
    </table>
  )
}
```

---

## Server-Side Patterns (Recommended for Large Datasets)

### Pattern 1: Server-Side Pagination with TanStack Query

**Generic REST API response shape:**

```typescript
// Expected API response from your Go backend (or any REST API)
interface PaginatedResponse<T> {
  data: T[]
  pagination: {
    page: number
    pageSize: number
    total: number
    pageCount: number
  }
}
```

**Client-Side Table with TanStack Query:**

```typescript
import { useReactTable, getCoreRowModel, PaginationState } from '@tanstack/react-table'
import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'

function ServerPaginatedTable() {
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
  })

  // TanStack Query fetches data
  const { data, isLoading } = useQuery({
    queryKey: ['users', pagination.pageIndex, pagination.pageSize],
    queryFn: async () => {
      const params = new URLSearchParams({
        page: pagination.pageIndex.toString(),
        page_size: pagination.pageSize.toString(),
      })
      const response = await fetch(`/api/users?${params}`)
      return response.json() as Promise<PaginatedResponse<User>>
    },
  })

  // TanStack Table manages display
  const table = useReactTable({
    data: data?.data ?? [],
    columns,
    getCoreRowModel: getCoreRowModel(),
    // Server-side pagination config
    manualPagination: true, // CRITICAL: Tell table pagination is manual
    pageCount: data?.pagination.pageCount ?? 0,
    state: { pagination },
    onPaginationChange: setPagination,
  })

  if (isLoading) return <div>Loading...</div>

  return (
    <div>
      <table>{/* render table */}</table>

      {/* Pagination controls */}
      <div>
        <button
          onClick={() => table.previousPage()}
          disabled={!table.getCanPreviousPage()}
        >
          Previous
        </button>
        <span>
          Page {table.getState().pagination.pageIndex + 1} of{' '}
          {table.getPageCount()}
        </span>
        <button
          onClick={() => table.nextPage()}
          disabled={!table.getCanNextPage()}
        >
          Next
        </button>
      </div>
    </div>
  )
}
```

### Pattern 2: Server-Side Filtering

**Client-Side:**

```typescript
const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([])

const { data } = useQuery({
  queryKey: ['users', columnFilters],
  queryFn: async () => {
    const search = columnFilters.find(f => f.id === 'search')?.value || ''
    const response = await fetch(`/api/users?search=${encodeURIComponent(search as string)}`)
    return response.json()
  },
})

const table = useReactTable({
  data: data?.data ?? [],
  columns,
  getCoreRowModel: getCoreRowModel(),
  manualFiltering: true, // CRITICAL: Server handles filtering
  state: { columnFilters },
  onColumnFiltersChange: setColumnFilters,
})
```

### Pattern 3: Server-Side Sorting

```typescript
const [sorting, setSorting] = useState<SortingState>([])

const { data } = useQuery({
  queryKey: ['users', pagination, sorting],  // Include sorting in key
  queryFn: async () => {
    const params = new URLSearchParams({
      page: pagination.pageIndex.toString(),
      page_size: pagination.pageSize.toString(),
    })
    if (sorting[0]) {
      params.set('sort_by', sorting[0].id)
      params.set('sort_order', sorting[0].desc ? 'desc' : 'asc')
    }
    return fetch(`/api/users?${params}`).then(r => r.json())
  }
})

const table = useReactTable({
  data: data?.data ?? [],
  columns,
  getCoreRowModel: getCoreRowModel(),
  manualSorting: true,
  state: { sorting },
  onSortingChange: setSorting,
})
```

---

## Virtualization for Large Datasets

For 1000+ rows, use TanStack Virtual to only render visible rows:

```typescript
import { useVirtualizer } from '@tanstack/react-virtual'
import { useRef } from 'react'

function VirtualizedTable() {
  const tableContainerRef = useRef<HTMLDivElement>(null)

  const table = useReactTable({
    data: largeDataset,
    columns,
    getCoreRowModel: getCoreRowModel(),
  })

  const { rows } = table.getRowModel()

  const rowVirtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => tableContainerRef.current,
    estimateSize: () => 50,  // Row height in px
    overscan: 10,            // Render 10 extra rows for smooth scrolling
  })

  return (
    <div ref={tableContainerRef} style={{ height: '600px', overflow: 'auto' }}>
      <table style={{ height: `${rowVirtualizer.getTotalSize()}px` }}>
        <thead>{/* header */}</thead>
        <tbody>
          {rowVirtualizer.getVirtualItems().map(virtualRow => {
            const row = rows[virtualRow.index]
            return (
              <tr
                key={row.id}
                style={{
                  position: 'absolute',
                  transform: `translateY(${virtualRow.start}px)`,
                  width: '100%',
                }}
              >
                {row.getVisibleCells().map(cell => (
                  <td key={cell.id}>{cell.renderValue()}</td>
                ))}
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
```

---

## Common Errors & Solutions

### Error 1: Infinite Re-Renders

**Problem:** Table re-renders infinitely, browser freezes.

**Cause:** `data` or `columns` references change on every render.

**Solution:** Always use `useMemo` or `useState`:

```typescript
// BAD: New array reference every render
const data = [{ id: 1 }]

// GOOD: Stable reference
const data = useMemo(() => [{ id: 1 }], [])

// ALSO GOOD: Define outside component
const STATIC_DATA = [{ id: 1 }]
```

### Error 2: TanStack Query + Table State Mismatch

**Problem:** Query refetches but pagination state not in sync, causing stale data.

**Solution:** Include ALL table state in query key:

```typescript
// BAD: Missing pagination in query key
const { data } = useQuery({
  queryKey: ['users'],  // Doesn't include page!
  queryFn: () => fetch(`/api/users?page=${pagination.pageIndex}`).then(r => r.json())
})

// GOOD: Complete query key
const { data } = useQuery({
  queryKey: ['users', pagination.pageIndex, pagination.pageSize, columnFilters, sorting],
  queryFn: () => {
    const params = new URLSearchParams({
      page: pagination.pageIndex.toString(),
      page_size: pagination.pageSize.toString(),
    })
    return fetch(`/api/users?${params}`).then(r => r.json())
  }
})
```

### Error 3: Server-Side Features Not Working

**Problem:** Pagination/filtering/sorting doesn't trigger API calls.

**Solution:** Set `manual*` flags to `true`:

```typescript
const table = useReactTable({
  data,
  columns,
  getCoreRowModel: getCoreRowModel(),
  // CRITICAL: Tell table these are server-side
  manualPagination: true,
  manualFiltering: true,
  manualSorting: true,
  pageCount: serverPageCount,  // Must provide total page count
})
```

### Error 4: TypeScript "Cannot Find Module" for Column Helper

**Solution:** Import from correct path:

```typescript
// BAD
import { createColumnHelper } from '@tanstack/table-core'

// GOOD
import { createColumnHelper } from '@tanstack/react-table'

const columnHelper = createColumnHelper<User>()
const columns = [
  columnHelper.accessor('name', {
    header: 'Name',
    cell: info => info.getValue(),  // Fully typed
  }),
]
```

### Error 5: Sorting Not Working with Server-Side

**Solution:** Include sorting in query key and API call. See Pattern 3 above.

### Error 6: Poor Performance with Large Datasets

**Solution:** Use virtualization (see above) or implement server-side pagination.

---

## Integration with TanStack Query

TanStack Table + TanStack Query is the recommended pattern:

```typescript
// Query handles data fetching + caching
const { data, isLoading } = useQuery({
  queryKey: ['users', tableState],
  queryFn: fetchUsers,
})

// Table handles display + interactions
const table = useReactTable({
  data: data?.data ?? [],
  columns,
  getCoreRowModel: getCoreRowModel(),
})
```

---

## Best Practices

### 1. Always Memoize Data and Columns
```typescript
const data = useMemo(() => [...], [dependencies])
const columns = useMemo(() => [...], [])
```

### 2. Use Server-Side for Large Datasets
- Client-side: <1000 rows
- Server-side: 1000+ rows or frequently changing data

### 3. Coordinate Query Keys with Table State
```typescript
queryKey: ['resource', pagination, filters, sorting]
```

### 4. Provide Loading States
```typescript
if (isLoading) return <TableSkeleton />
if (error) return <ErrorMessage error={error} />
```

### 5. Use Column Helper for Type Safety
```typescript
const columnHelper = createColumnHelper<YourType>()
const columns = [
  columnHelper.accessor('field', { /* fully typed */ })
]
```

### 6. Virtualize Large Client-Side Tables
```typescript
if (data.length > 1000) {
  // Use TanStack Virtual (see example above)
}
```

### 7. Control Only the State You Need

Keep `sorting`, `pagination`, `filters`, `visibility`, `pinning`, `order`, `selection` in controlled state when you must persist or sync.

Avoid controlling `columnSizingInfo` unless persisting drag state — it triggers frequent updates and hurts performance.

---

## Column Configuration Patterns

```typescript
import { createColumnHelper } from '@tanstack/react-table'

const columnHelper = createColumnHelper<User>()

const columns = [
  // Accessor column (data)
  columnHelper.accessor('name', {
    header: 'Name',
    cell: info => info.getValue(),
    enableSorting: true,
    enableColumnFilter: true,
  }),

  // Display column (no data accessor)
  columnHelper.display({
    id: 'actions',
    header: 'Actions',
    cell: props => <RowActions row={props.row} />,
  }),

  // Grouped column (nested headers)
  columnHelper.group({
    header: 'Personal Info',
    columns: [
      columnHelper.accessor('firstName', { header: 'First Name' }),
      columnHelper.accessor('lastName', { header: 'Last Name' }),
    ],
  }),
]
```

---

## Controlled State Patterns

```typescript
// Column visibility
const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({
  id: false,  // Hide ID column by default
})

// Row selection
const [rowSelection, setRowSelection] = useState<RowSelectionState>({})

// Column pinning
const [columnPinning, setColumnPinning] = useState<ColumnPinningState>({
  left: ['name'],  // Freeze name column on left
})

const table = useReactTable({
  data,
  columns,
  getCoreRowModel: getCoreRowModel(),
  state: {
    columnVisibility,
    rowSelection,
    columnPinning,
    pagination,
    sorting,
    columnFilters,
  },
  onColumnVisibilityChange: setColumnVisibility,
  onRowSelectionChange: setRowSelection,
  onColumnPinningChange: setColumnPinning,
  onPaginationChange: setPagination,
  onSortingChange: setSorting,
  onColumnFiltersChange: setColumnFilters,
})
```

---

## shadcn/ui Integration

```typescript
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/table'

function StyledTable() {
  const table = useReactTable({ /* config */ })

  return (
    <Table>
      <TableHeader>
        {table.getHeaderGroups().map(headerGroup => (
          <TableRow key={headerGroup.id}>
            {headerGroup.headers.map(header => (
              <TableHead key={header.id}>
                {header.column.columnDef.header as string}
              </TableHead>
            ))}
          </TableRow>
        ))}
      </TableHeader>
      <TableBody>
        {table.getRowModel().rows.map(row => (
          <TableRow key={row.id}>
            {row.getVisibleCells().map(cell => (
              <TableCell key={cell.id}>
                {cell.renderValue() as string}
              </TableCell>
            ))}
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}
```

---

## Further Reading

- **Official Docs:** https://tanstack.com/table/latest
- **TanStack Virtual:** https://tanstack.com/virtual/latest
- **GitHub:** https://github.com/TanStack/table

---

**Skill Version:** 1.0.0
**Library Version:** @tanstack/react-table v8.21.3
