# Table Component

A production-ready, feature-rich data table component built with **@tanstack/react-table** and **@tanstack/react-virtual**.

## Features

- ✅ **Virtual Scrolling** - Handle thousands of rows with excellent performance
- ✅ **Sorting** - Single and multi-column sorting (client & server-side)
- ✅ **Filtering** - Column filters and global search
- ✅ **Pagination** - Client-side, server-side, or infinite scroll
- ✅ **Row Selection** - Single and multi-row selection
- ✅ **Column Resizing** - Drag to resize columns
- ✅ **Column Reordering** - Drag and drop column reordering
- ✅ **Column Visibility** - Toggle column visibility
- ✅ **Keyboard Navigation** - Arrow keys and Vim bindings (j/k)
- ✅ **Export** - Export to CSV and JSON
- ✅ **Persistence** - Save column state to localStorage
- ✅ **Loading States** - Skeleton loaders and custom loading states
- ✅ **Empty States** - Customizable empty state
- ✅ **Dark Mode** - Full dark mode support
- ✅ **TypeScript** - Full type safety
- ✅ **Accessible** - ARIA labels and keyboard support

## Installation

The table component is part of `@unkey/ui`:

```bash
pnpm add @unkey/ui
```

## Basic Usage

```tsx
import { Table, type ColumnDef } from "@unkey/ui";

interface User {
  id: string;
  name: string;
  email: string;
  role: string;
}

const columns: ColumnDef<User>[] = [
  {
    accessorKey: "name",
    header: "Name",
  },
  {
    accessorKey: "email",
    header: "Email",
  },
  {
    accessorKey: "role",
    header: "Role",
  },
];

function UsersTable() {
  const [data, setData] = useState<User[]>([]);

  return (
    <Table
      data={data}
      columns={columns}
      enableSorting
      enableGlobalFilter
      enableVirtualization
    />
  );
}
```

## Column Definitions

Columns use the standard `@tanstack/react-table` `ColumnDef` type:

```tsx
const columns: ColumnDef<User>[] = [
  {
    accessorKey: "name",
    header: "Name",
    cell: ({ row }) => (
      <div className="font-medium">{row.getValue("name")}</div>
    ),
    size: 200,
    minSize: 100,
    maxSize: 400,
    enableSorting: true,
    enableResizing: true,
  },
  {
    id: "actions",
    header: "Actions",
    cell: ({ row }) => (
      <button onClick={() => handleEdit(row.original)}>Edit</button>
    ),
    enableSorting: false,
  },
];
```

## Features

### Virtualization

Virtual scrolling is enabled by default for optimal performance:

```tsx
<Table
  data={largeDataset}
  columns={columns}
  enableVirtualization={true}
  estimateSize={52} // Row height in pixels
  overscan={5} // Number of rows to render outside viewport
/>
```

### Sorting

Enable sorting (client-side or server-side):

```tsx
// Client-side sorting
<Table
  data={data}
  columns={columns}
  enableSorting={true}
/>

// Server-side sorting (controlled)
const [sorting, setSorting] = useState<SortingState>([]);

<Table
  data={data}
  columns={columns}
  enableSorting={true}
  manualSorting={true}
  sorting={sorting}
  onSortingChange={setSorting}
/>
```

### Filtering

Enable column filters and global search:

```tsx
<Table
  data={data}
  columns={columns}
  enableFiltering={true}
  enableGlobalFilter={true}
/>
```

### Pagination

Choose between client-side pagination, server-side pagination, or infinite scroll:

```tsx
// Client-side pagination
<Table
  data={data}
  columns={columns}
  enablePagination={true}
/>

// Infinite scroll
<Table
  data={data}
  columns={columns}
  enableInfiniteScroll={true}
  hasMore={hasMore}
  onFetchMore={loadMore}
  isLoadingMore={isLoadingMore}
/>
```

### Row Selection

Enable single or multi-row selection:

```tsx
const [rowSelection, setRowSelection] = useState({});

<Table
  data={data}
  columns={columns}
  enableRowSelection={true}
  enableMultiRowSelection={true}
  rowSelection={rowSelection}
  onRowSelectionChange={setRowSelection}
/>
```

### Column Resizing

Enable drag-to-resize columns:

```tsx
<Table
  data={data}
  columns={columns}
  enableColumnResizing={true}
/>
```

### Column Visibility

Toggle column visibility:

```tsx
<Table
  data={data}
  columns={columns}
  enableColumnVisibility={true}
/>
```

### Keyboard Navigation

Navigate with arrow keys or Vim bindings (j/k):

```tsx
<Table
  data={data}
  columns={columns}
  enableKeyboardNavigation={true}
  onRowClick={(row) => console.log("Selected:", row)}
/>
```

**Keyboard shortcuts:**
- `↑` / `k` - Move up
- `↓` / `j` - Move down
- `Enter` - Select row
- `Space` - Toggle selection
- `Escape` - Clear selection

### Export

Export table data to CSV or JSON:

```tsx
<Table
  data={data}
  columns={columns}
  enableExport={true}
  exportFilename="users"
  exportFormats={["csv", "json"]}
/>
```

### Persistence

Persist column state to localStorage:

```tsx
<Table
  data={data}
  columns={columns}
  persistenceKey="users-table"
  persistColumnOrder={true}
  persistColumnSizing={true}
  persistColumnVisibility={true}
/>
```

### Loading & Empty States

Customize loading and empty states:

```tsx
<Table
  data={data}
  columns={columns}
  isLoading={isLoading}
  emptyState={
    <div>
      <h3>No users found</h3>
      <button onClick={createUser}>Create User</button>
    </div>
  }
  loadingState={<CustomLoader />}
  skeletonRows={10}
/>
```

### Styling

Customize table appearance:

```tsx
<Table
  data={data}
  columns={columns}
  layoutMode="grid" // or "classic"
  rowBorders={true}
  stickyHeader={true}
  maxHeight="600px"
  className="custom-table"
  rowClassName={(row) => row.role === "admin" ? "bg-blue-50" : ""}
/>
```

## API Reference

### TableProps

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `data` | `TData[]` | **required** | Array of data objects |
| `columns` | `ColumnDef<TData>[]` | **required** | Column definitions |
| `enableSorting` | `boolean` | `true` | Enable sorting |
| `enableFiltering` | `boolean` | `false` | Enable column filters |
| `enableGlobalFilter` | `boolean` | `false` | Enable global search |
| `enablePagination` | `boolean` | `false` | Enable pagination |
| `enableRowSelection` | `boolean` | `false` | Enable row selection |
| `enableMultiRowSelection` | `boolean` | `true` | Enable multi-row selection |
| `enableColumnResizing` | `boolean` | `false` | Enable column resizing |
| `enableColumnReordering` | `boolean` | `false` | Enable column reordering |
| `enableColumnVisibility` | `boolean` | `false` | Enable column visibility toggle |
| `enableVirtualization` | `boolean` | `true` | Enable virtual scrolling |
| `enableInfiniteScroll` | `boolean` | `false` | Enable infinite scroll |
| `enableExport` | `boolean` | `false` | Enable data export |
| `enableKeyboardNavigation` | `boolean` | `true` | Enable keyboard navigation |
| `manualSorting` | `boolean` | `false` | Use server-side sorting |
| `manualFiltering` | `boolean` | `false` | Use server-side filtering |
| `manualPagination` | `boolean` | `false` | Use server-side pagination |
| `estimateSize` | `number` | `52` | Estimated row height (px) |
| `overscan` | `number` | `5` | Rows to render outside viewport |
| `layoutMode` | `"classic" \| "grid"` | `"grid"` | Table layout mode |
| `rowBorders` | `boolean` | `true` | Show row borders |
| `stickyHeader` | `boolean` | `false` | Sticky table header |
| `maxHeight` | `string \| number` | `"600px"` | Max table height |
| `persistenceKey` | `string` | - | Key for localStorage |
| `persistColumnOrder` | `boolean` | `false` | Persist column order |
| `persistColumnSizing` | `boolean` | `false` | Persist column sizes |
| `persistColumnVisibility` | `boolean` | `false` | Persist column visibility |

See [types.ts](./types.ts) for complete API reference.

## Advanced Examples

### Server-Side Data with Infinite Scroll

```tsx
function ServerSideTable() {
  const [data, setData] = useState<User[]>([]);
  const [hasMore, setHasMore] = useState(true);
  const [cursor, setCursor] = useState<string | null>(null);
  const [isLoadingMore, setIsLoadingMore] = useState(false);

  const loadMore = async () => {
    setIsLoadingMore(true);
    const response = await fetch(`/api/users?cursor=${cursor}`);
    const { users, nextCursor } = await response.json();

    setData((prev) => [...prev, ...users]);
    setCursor(nextCursor);
    setHasMore(!!nextCursor);
    setIsLoadingMore(false);
  };

  return (
    <Table
      data={data}
      columns={columns}
      enableInfiniteScroll
      hasMore={hasMore}
      onFetchMore={loadMore}
      isLoadingMore={isLoadingMore}
      enableVirtualization
    />
  );
}
```

### Custom Cell Renderers

```tsx
const columns: ColumnDef<User>[] = [
  {
    accessorKey: "avatar",
    header: "Avatar",
    cell: ({ row }) => (
      <img
        src={row.getValue("avatar")}
        alt={row.getValue("name")}
        className="w-8 h-8 rounded-full"
      />
    ),
  },
  {
    accessorKey: "status",
    header: "Status",
    cell: ({ row }) => {
      const status = row.getValue("status");
      return (
        <span className={`badge ${status === "active" ? "bg-green-500" : "bg-gray-500"}`}>
          {status}
        </span>
      );
    },
  },
];
```

### Row Actions

```tsx
const columns: ColumnDef<User>[] = [
  // ... other columns
  {
    id: "actions",
    header: "Actions",
    cell: ({ row }) => (
      <div className="flex gap-2">
        <button onClick={() => handleEdit(row.original)}>Edit</button>
        <button onClick={() => handleDelete(row.original.id)}>Delete</button>
      </div>
    ),
  },
];
```

## Architecture

The table component is built on top of:

- **[@tanstack/react-table](https://tanstack.com/table)** - Headless table state management
- **[@tanstack/react-virtual](https://tanstack.com/virtual)** - Virtual scrolling for performance

This architecture provides:
- 🎯 Full control over rendering and styling
- 🚀 Excellent performance with large datasets
- 🔧 All features from @tanstack/react-table
- 🎨 Customizable with Tailwind CSS

## Performance Tips

1. **Enable virtualization** for tables with > 100 rows
2. **Use `useMemo`** for column definitions
3. **Use `getRowId`** for stable row keys
4. **Enable server-side operations** for very large datasets
5. **Limit `overscan`** if rows are expensive to render

## TypeScript

The component is fully typed:

```tsx
import { Table, type ColumnDef, type TableProps } from "@unkey/ui";

interface User {
  id: string;
  name: string;
  email: string;
}

const columns: ColumnDef<User>[] = [...];

const tableProps: TableProps<User> = {
  data: users,
  columns,
  enableSorting: true,
};
```

## Contributing

The table component is located in `internal/ui/src/table/`:

```
table/
├── index.tsx           # Main Table component
├── types.ts            # TypeScript definitions
├── constants.ts        # Default configuration
├── hooks/              # Custom hooks
│   ├── useTableState.ts
│   ├── useKeyboardNavigation.ts
│   └── useTableVirtualizer.ts
├── components/         # Sub-components
│   ├── table-header.tsx
│   ├── table-body.tsx
│   ├── toolbar.tsx
│   ├── empty-state.tsx
│   └── skeleton.tsx
└── utils/              # Utility functions
    ├── export.ts
    └── storage.ts
```

## License

AGPL-3.0
