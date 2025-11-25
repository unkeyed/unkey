import type {
  Column,
  ColumnDef,
  ColumnFiltersState,
  ColumnOrderState,
  ColumnSizingState,
  OnChangeFn,
  PaginationState,
  Row,
  RowSelectionState,
  SortingState,
  Table as TanstackTable,
  VisibilityState,
} from "@tanstack/react-table";
import type { VirtualItem, Virtualizer } from "@tanstack/react-virtual";
import type React from "react";

// ============================================================================
// Core Table Types
// ============================================================================

export interface TableProps<TData> {
  // Data
  data: TData[];
  columns: ColumnDef<TData, unknown>[];

  // Table state (controlled)
  sorting?: SortingState;
  onSortingChange?: OnChangeFn<SortingState>;
  columnFilters?: ColumnFiltersState;
  onColumnFiltersChange?: OnChangeFn<ColumnFiltersState>;
  globalFilter?: string;
  onGlobalFilterChange?: OnChangeFn<string>;
  pagination?: PaginationState;
  onPaginationChange?: OnChangeFn<PaginationState>;
  rowSelection?: RowSelectionState;
  onRowSelectionChange?: OnChangeFn<RowSelectionState>;
  columnVisibility?: VisibilityState;
  onColumnVisibilityChange?: OnChangeFn<VisibilityState>;
  columnOrder?: ColumnOrderState;
  onColumnOrderChange?: OnChangeFn<ColumnOrderState>;
  columnSizing?: ColumnSizingState;
  onColumnSizingChange?: OnChangeFn<ColumnSizingState>;

  // Features
  enableSorting?: boolean;
  enableFiltering?: boolean;
  enableGlobalFilter?: boolean;
  enablePagination?: boolean;
  enableRowSelection?: boolean | ((row: Row<TData>) => boolean);
  enableMultiRowSelection?: boolean;
  enableColumnResizing?: boolean;
  enableColumnReordering?: boolean;
  enableColumnVisibility?: boolean;
  enableVirtualization?: boolean;
  enableInfiniteScroll?: boolean;
  enableRealtimeData?: boolean;
  enableExport?: boolean;
  enableKeyboardNavigation?: boolean;

  // Data mode
  manualSorting?: boolean;
  manualFiltering?: boolean;
  manualPagination?: boolean;

  // Data fetching
  onFetchMore?: () => void;
  hasMore?: boolean;
  isLoading?: boolean;
  isLoadingMore?: boolean;
  totalCount?: number;

  // Virtualization options
  estimateSize?: number;
  overscan?: number;
  virtualPaddingStart?: number;
  virtualPaddingEnd?: number;

  // Layout & styling
  layoutMode?: "classic" | "grid";
  rowBorders?: boolean;
  stickyHeader?: boolean;
  maxHeight?: string | number;
  className?: string;
  rowClassName?: string | ((row: TData) => string);

  // Empty & loading states
  emptyState?: React.ReactNode;
  loadingState?: React.ReactNode;
  skeletonRows?: number;

  // Callbacks
  onRowClick?: (row: TData) => void;
  onRowDoubleClick?: (row: TData) => void;
  getRowId?: (row: TData, index: number) => string;

  // Persistence
  persistenceKey?: string;
  persistColumnOrder?: boolean;
  persistColumnSizing?: boolean;
  persistColumnVisibility?: boolean;

  // Export options
  exportFilename?: string;
  exportFormats?: ExportFormat[];

  // Realtime data
  realtimeData?: TData[];
  realtimeSeparator?: boolean;

  // Accessibility
  ariaLabel?: string;
  ariaDescribedBy?: string;
}

// ============================================================================
// Filter Types
// ============================================================================

export type FilterType = "text" | "select" | "number" | "date" | "dateRange" | "boolean";

export interface FilterOption {
  label: string;
  value: string;
}

export interface ColumnFilterConfig {
  type: FilterType;
  placeholder?: string;
  options?: FilterOption[];
  enableMultiSelect?: boolean;
  minValue?: number;
  maxValue?: number;
  minDate?: Date;
  maxDate?: Date;
}

export interface GlobalFilterProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  className?: string;
}

export interface ColumnFilterProps<TData = unknown> {
  column: Column<TData, unknown>;
  config: ColumnFilterConfig;
}

// ============================================================================
// Export Types
// ============================================================================

export type ExportFormat = "csv" | "json" | "xlsx";

export interface ExportOptions {
  format: ExportFormat;
  filename?: string;
  includeHeaders?: boolean;
  selectedRowsOnly?: boolean;
  visibleColumnsOnly?: boolean;
}

export interface ExportConfig {
  enableCSV?: boolean;
  enableJSON?: boolean;
  enableXLSX?: boolean;
  defaultFilename?: string;
  onExport?: (options: ExportOptions) => void;
}

// ============================================================================
// Pagination Types
// ============================================================================

export type PaginationMode = "client" | "server" | "infinite" | "cursor";

export interface PaginationConfig {
  mode: PaginationMode;
  pageSize: number;
  pageSizeOptions?: number[];
  showPageSizeSelector?: boolean;
  showPageInfo?: boolean;
  showFirstLast?: boolean;
}

// ============================================================================
// Sorting Types
// ============================================================================

export type SortDirection = "asc" | "desc" | false;

export interface SortConfig {
  enableMultiSort?: boolean;
  maxMultiSortColCount?: number;
  sortDescFirst?: boolean;
  isMultiSortEvent?: (e: unknown) => boolean;
}

// ============================================================================
// Selection Types
// ============================================================================

export interface SelectionConfig<TData = unknown> {
  enableMultiRowSelection?: boolean;
  enableSubRowSelection?: boolean;
  enableSelectAll?: boolean;
  selectAllMode?: "all" | "page";
  onSelectionChange?: (selectedRows: TData[]) => void;
}

// ============================================================================
// Keyboard Navigation Types
// ============================================================================

export type KeyboardAction =
  | "moveUp"
  | "moveDown"
  | "moveLeft"
  | "moveRight"
  | "selectRow"
  | "toggleSelection"
  | "selectAll"
  | "clearSelection"
  | "escape";

export interface KeyboardConfig {
  enableVimBindings?: boolean;
  customKeyMap?: Partial<Record<string, KeyboardAction>>;
  onKeyAction?: (action: KeyboardAction, event: KeyboardEvent) => void;
}

// ============================================================================
// Realtime Data Types
// ============================================================================

export interface RealtimeConfig {
  enabled?: boolean;
  pollingInterval?: number;
  separatorLabel?: string;
  highlightNew?: boolean;
  highlightDuration?: number;
  maxRealtimeItems?: number;
}

// ============================================================================
// Column Resizing Types
// ============================================================================

export interface ColumnResizeConfig {
  enabled?: boolean;
  mode?: "onChange" | "onEnd";
  minWidth?: number;
  maxWidth?: number;
  defaultWidth?: number;
}

// ============================================================================
// Column Reordering Types
// ============================================================================

export interface ColumnReorderConfig {
  enabled?: boolean;
  onReorder?: (newOrder: string[]) => void;
}

// ============================================================================
// Loading State Types
// ============================================================================

export interface LoadingConfig {
  showSkeleton?: boolean;
  skeletonRows?: number;
  customSkeleton?: React.ReactNode;
  loadingOverlay?: boolean;
  loadingMessage?: string;
}

// ============================================================================
// Empty State Types
// ============================================================================

export interface EmptyStateConfig {
  title?: string;
  description?: string;
  icon?: React.ReactNode;
  action?: {
    label: string;
    onClick: () => void;
  };
}

// ============================================================================
// Utility Types
// ============================================================================

export interface TableInstance<TData> {
  table: TanstackTable<TData>;
  virtualizer?: Virtualizer<HTMLDivElement, Element>;
  getSelectedRows: () => TData[];
  clearSelection: () => void;
  selectAll: () => void;
  exportData: (options: ExportOptions) => void;
  refresh: () => void;
}

export interface VirtualRow<TData = unknown> {
  virtualRow: VirtualItem;
  row: Row<TData>;
}

// ============================================================================
// Component Prop Types
// ============================================================================

export interface TableHeaderProps<TData = unknown> {
  table: TanstackTable<TData>;
  stickyHeader?: boolean;
  enableColumnResizing?: boolean;
  enableColumnReordering?: boolean;
}

export interface TableBodyProps<TData> {
  table: TanstackTable<TData>;
  virtualizer?: Virtualizer<HTMLDivElement, Element>;
  virtualRows?: VirtualItem[];
  isLoading?: boolean;
  emptyState?: React.ReactNode;
  loadingState?: React.ReactNode;
  onRowClick?: (row: TData) => void;
  rowClassName?: string | ((row: TData) => string);
}

export interface TableFooterProps<TData = unknown> {
  table: TanstackTable<TData>;
  hasMore?: boolean;
  isLoadingMore?: boolean;
  onLoadMore?: () => void;
  totalCount?: number;
}

export interface TableToolbarProps<TData = unknown> {
  table: TanstackTable<TData>;
  enableGlobalFilter?: boolean;
  enableColumnVisibility?: boolean;
  enableExport?: boolean;
  exportConfig?: ExportConfig;
}

// ============================================================================
// Storage Types
// ============================================================================

export interface TableStorageState {
  columnOrder?: ColumnOrderState;
  columnSizing?: ColumnSizingState;
  columnVisibility?: VisibilityState;
}

export interface StorageConfig {
  key: string;
  storage?: Storage;
  debounceMs?: number;
}
