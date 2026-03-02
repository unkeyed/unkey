# DataTable Component

A table component built with TanStack Table v8, designed for paginated and real-time entity lists in the dashboard.

## Features

- **Cursor-based pagination**: `PaginationFooter` with minimizable floating panel
- **Load-more pagination**: `LoadMoreFooter` for infinite-scroll patterns
- **Real-time data**: automatic merging with a "Live" separator UI
- **Sorting**: 3-state (null → asc → desc → null), single column
- **Row selection**: optional multi-select with checkboxes
- **Keyboard navigation**: Arrow / j / k / Escape
- **Layout modes**: Grid (no row spacing) and Classic (spacer rows)
- **Skeleton rows**: customizable per table via `renderSkeletonRow`
- **Empty states**: customizable via `emptyState` prop
- **Responsive height**: auto-calculated via `ResizeObserver`, fixed 400px on mobile
- **Full TypeScript**: all props and column definitions are typed

## Import

```typescript
import { DataTable, type DataTableColumnDef } from "@/components/data-table";
```

## Basic Usage

### Grid layout with cursor-based pagination (root keys pattern)

```typescript
import { DataTable, createRootKeyColumns } from "@/components/data-table";
import { PaginationFooter } from "@/components/data-table/components/footer/pagination-footer";
import { renderRootKeySkeletonRow } from "@/components/data-table/components/skeletons/render-root-key-skeleton-row";
import { useRootKeysListPaginated } from "@/components/data-table/hooks/rootkey/use-root-keys-list-query";

const TABLE_CONFIG = {
  rowHeight: 40,
  layout: "grid" as const,
  rowBorders: true,
  containerPadding: "px-0",
};

export function RootKeysList() {
  const { rootKeys, isLoading, isFetching, isPending, totalCount, onPageChange, page, pageSize, totalPages } =
    useRootKeysListPaginated();
  const [selectedKey, setSelectedKey] = useState<RootKey | null>(null);

  const columns = useMemo(
    () => createRootKeyColumns({ selectedRootKeyId: selectedKey?.id, onEditKey: handleEdit }),
    [selectedKey, handleEdit],
  );

  return (
    <>
      <DataTable
        data={rootKeys}
        columns={columns}
        getRowId={(key) => key.id}
        isLoading={isLoading || isFetching}
        onRowClick={setSelectedKey}
        selectedItem={selectedKey}
        config={TABLE_CONFIG}
        renderSkeletonRow={renderRootKeySkeletonRow}
        enableSorting
      />
      <PaginationFooter
        page={page}
        pageSize={pageSize}
        totalPages={totalPages}
        totalCount={totalCount}
        onPageChange={onPageChange}
        itemLabel="root keys"
        hide={isPending}
      />
    </>
  );
}
```

### Classic layout with load-more and real-time data

```typescript
import { DataTable, type DataTableColumnDef } from "@/components/data-table";

const columns: DataTableColumnDef<Log>[] = [
  {
    id: "timestamp",
    accessorKey: "timestamp",
    header: "Time",
    meta: { width: 150 },
    cell: ({ row }) => <TimestampInfo value={row.original.timestamp} />,
  },
  {
    id: "status",
    accessorKey: "status",
    header: "Status",
    meta: { width: 80 },
    cell: ({ row }) => <StatusCell status={row.original.status} />,
  },
];

export function LogsTable() {
  const { realtimeLogs } = useRealtimeLogs();
  const { logs, fetchNextPage, hasNextPage, isFetchingNextPage } = useHistoricalLogs();

  return (
    <DataTable
      data={logs}
      realtimeData={realtimeLogs}
      columns={columns}
      getRowId={(log) => log.id}
      config={{ layout: "classic", rowHeight: 32 }}
      loadMoreFooterProps={{
        onLoadMore: fetchNextPage,
        hasMore: hasNextPage,
        isFetchingNextPage,
        itemLabel: "logs",
      }}
    />
  );
}
```

## Props

### Required

| Prop | Type | Description |
|------|------|-------------|
| `data` | `TData[]` | Historical / paginated rows |
| `columns` | `DataTableColumnDef<TData>[]` | TanStack column definitions |
| `getRowId` | `(row: TData) => string` | Unique row identifier |

### Data

| Prop | Type | Description |
|------|------|-------------|
| `realtimeData` | `TData[]` | Real-time rows prepended above a separator |

### Interaction

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `onRowClick` | `(row: TData \| null) => void` | — | Row click handler; called with `null` on Escape |
| `onRowMouseEnter` | `(row: TData) => void` | — | Row hover start |
| `onRowMouseLeave` | `() => void` | — | Row hover end |
| `selectedItem` | `TData \| null` | — | Highlighted row (pass the same object you get from `onRowClick`) |

### Sorting

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `enableSorting` | `boolean` | `true` | Enable column sorting |
| `sorting` | `SortingState` | — | Controlled sort state |
| `onSortingChange` | `OnChangeFn<SortingState>` | — | Sort state change handler |

### Row selection

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `enableRowSelection` | `boolean` | `false` | Show checkboxes |
| `rowSelection` | `RowSelectionState` | — | Controlled selection state |
| `onRowSelectionChange` | `OnChangeFn<RowSelectionState>` | — | Selection change handler |

### Inline pagination (within table scroll area)

| Prop | Type | Description |
|------|------|-------------|
| `page` | `number` | Current page (1-indexed) |
| `pageSize` | `number` | Rows per page |
| `totalPages` | `number` | Total page count |
| `onPageChange` | `(page: number) => void` | Page change handler |

> All four must be provided together to render the inline `Pagination` bar. For the floating `PaginationFooter`, render it separately outside `DataTable`.

### Loading and UI

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `isLoading` | `boolean` | `false` | Renders skeleton rows instead of data |
| `renderSkeletonRow` | `(props) => ReactNode` | `SkeletonRow` | Custom per-column skeleton renderer |
| `emptyState` | `ReactNode` | `<EmptyState />` | Rendered when data and realtimeData are both empty |
| `loadMoreFooterProps` | `LoadMoreFooterProps` | — | Activates the floating load-more footer |
| `rowClassName` | `(row: TData) => string` | — | Per-row className |
| `selectedClassName` | `(row: TData, isSelected: boolean) => string` | — | Per-row className when selected |
| `fixedHeight` | `number` | auto-calculated | Override the auto height calculation |
| `enableKeyboardNav` | `boolean` | `true` | Arrow / j / k / Escape navigation |
| `config` | `Partial<DataTableConfig>` | see below | Layout and dimension overrides |

## Configuration

```typescript
interface DataTableConfig {
  rowHeight: number;        // Default: 36
  headerHeight: number;     // Default: 40
  rowSpacing: number;       // Default: 4 — gap between rows in classic layout

  layout: "grid" | "classic"; // Default: "classic"
  rowBorders: boolean;      // Default: false — horizontal divider between rows
  containerPadding: string; // Default: "px-2"
  tableLayout: "fixed" | "auto"; // Default: "fixed"

  loadingRows: number;      // Default: 10 — skeleton row count while isLoading
}
```

**Grid layout** — rows sit directly adjacent, no spacers. Used for entity lists (root keys, API keys).

**Classic layout** — a spacer `<tr>` is inserted before each data row for visual breathing room. Used for log tables.

## Column Definition

### Width options

```typescript
meta: { width: 150 }             // Fixed pixels
meta: { width: "25%" }           // Percentage of table width
meta: { width: "165px" }         // Any CSS value
meta: { width: "auto" }          // Browser auto sizing
meta: { width: "min" }           // min-content
meta: { width: "1fr" }           // Flex grow
meta: { width: { min: 100, max: 300 } } // Clamp range
meta: { width: { flex: 2 } }     // Flex ratio
```

### Per-column styling

```typescript
{
  id: "name",
  accessorKey: "name",
  header: "Name",
  meta: {
    headerClassName: "pl-4",
    cellClassName: "font-medium text-gray-12",
  },
}
```

### Sortable column with `SortableHeader`

```typescript
import { SortableHeader } from "@/components/data-table";

{
  id: "created_at",
  accessorKey: "createdAt",
  sortingFn: "alphanumeric",
  header: ({ header }) => (
    <SortableHeader header={header}>Created At</SortableHeader>
  ),
}
```

## Cell Components

All exported from `@/components/data-table`:

| Component | Props | Use for |
|-----------|-------|---------|
| `CheckboxCell` | `row`, `table` | Row selection checkbox |
| `CheckboxHeaderCell` | `table` | Select-all header checkbox |
| `StatusCell` | `status`, `icon?`, `label?` | Status badge with icon |
| `BadgeCell` | `value`, `variant?` | Generic badge |
| `CopyCell` | `value`, `label?` | Click-to-copy value |
| `HiddenValueCell` | `value`, `title`, `selected?` | Obfuscated key with reveal |
| `TimestampCell` | `timestamp`, `format?` | Relative / absolute / both |
| `LastUpdatedCell` | `lastUpdated`, `isSelected?` | Relative time with tooltip |
| `AssignedItemsCell` | `permissionSummary`, `isSelected?` | Permission count badge |
| `RootKeyNameCell` | `name`, `isSelected?` | Name with selection indicator |

## Footers

### `PaginationFooter`

Floating fixed footer with page controls. Minimizes to a compact pill at the bottom-right.

```typescript
import { PaginationFooter } from "@/components/data-table/components/footer/pagination-footer";

<PaginationFooter
  page={page}
  pageSize={pageSize}
  totalPages={totalPages}
  totalCount={totalCount}
  onPageChange={onPageChange}
  itemLabel="root keys"   // Label in "Viewing 1–50 of 200 root keys"
  hide={isPending}        // Hide during page transitions
  headerContent={<FilterBar />} // Optional content above the controls
/>
```

### `LoadMoreFooter`

Floating fixed footer with a "Load more" button. Used with infinite-scroll data.

```typescript
<DataTable
  data={data}
  columns={columns}
  getRowId={(item) => item.id}
  loadMoreFooterProps={{
    onLoadMore: fetchNextPage,
    hasMore: hasNextPage,
    isFetchingNextPage,
    itemLabel: "logs",
    buttonText: "Load more logs",
  }}
/>
```

## Pagination Hook

`useRootKeysListPaginated` implements cursor-based pagination for the root keys table. It stores page cursors in a ref so navigating back to a previous page is instant (uses the tRPC cache).

```typescript
const {
  rootKeys,     // TData[] for the current page
  isLoading,    // Initial load
  isFetching,   // Background refetch
  isPending,    // useTransition — page change in flight
  page,         // Current page number (1-indexed)
  pageSize,     // Rows per page
  totalPages,
  totalCount,
  onPageChange, // (newPage: number) => void
} = useRootKeysListPaginated(pageSize?: number);
```

Navigation to a page is only allowed if the cursor for that page has already been cached. Pages must be visited in order to accumulate cursors — random-access skipping is not possible.

## Real-time Data

Pass `realtimeData` alongside `data`. The component automatically:

1. Prepends real-time rows at the top
2. Deduplicates any rows that appear in both arrays (by `getRowId`)
3. Inserts a `<RealtimeSeparator />` at the boundary

```typescript
<DataTable
  data={historicalLogs}
  realtimeData={newLogs}
  getRowId={(log) => log.id}
  columns={columns}
/>
```

## Keyboard Navigation

When `enableKeyboardNav` is true (default):

| Key | Action |
|-----|--------|
| `ArrowDown` / `j` | Focus and click the next row |
| `ArrowUp` / `k` | Focus and click the previous row |
| `Escape` | Calls `onRowClick(null)` and blurs focus |

## Column ID Constants

When building custom skeleton renderers or referencing root key column IDs, use the exported constants to stay in sync with column definitions:

```typescript
import { ROOT_KEY_COLUMN_IDS } from "@/components/data-table";

// ROOT_KEY_COLUMN_IDS.ROOT_KEY   → "root_key"
// ROOT_KEY_COLUMN_IDS.KEY        → "key"
// ROOT_KEY_COLUMN_IDS.PERMISSIONS → "permissions"
// ROOT_KEY_COLUMN_IDS.CREATED_AT → "created_at"
// ROOT_KEY_COLUMN_IDS.LAST_UPDATED → "last_updated"
// ROOT_KEY_COLUMN_IDS.ACTION     → "action"
```

## Custom Skeleton Renderer

Provide `renderSkeletonRow` to render per-column skeleton content. The function receives the column list and `rowHeight` and must return `<td>` elements:

```typescript
import { ROOT_KEY_COLUMN_IDS } from "@/components/data-table";

const renderSkeletonRow = ({ columns, rowHeight }) =>
  columns.map((column) => (
    <td key={column.id} style={{ height: `${rowHeight}px` }}>
      {column.id === ROOT_KEY_COLUMN_IDS.ROOT_KEY && <RootKeyColumnSkeleton />}
      {column.id === ROOT_KEY_COLUMN_IDS.KEY && <KeyColumnSkeleton />}
    </td>
  ));

<DataTable renderSkeletonRow={renderSkeletonRow} ... />
```

If `renderSkeletonRow` is not provided, `<SkeletonRow>` renders a uniform shimmer across all columns.
