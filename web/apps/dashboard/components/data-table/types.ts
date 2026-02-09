import type { ColumnDef, RowSelectionState, SortingState } from "@tanstack/react-table";
import type { ReactNode } from "react";

/**
 * Column width configuration options
 */
export type ColumnWidth =
  | number // Fixed pixels
  | string // CSS: "165px", "15%"
  | "auto"
  | "min"
  | "1fr" // Keywords
  | { min: number; max: number } // Range
  | { flex: number }; // Flex ratio

/**
 * Custom column metadata
 */
export interface DataTableColumnMeta {
  // Styling
  headerClassName?: string;
  cellClassName?: string;

  // Width configuration
  width?: ColumnWidth;

  // Display options
  isCopyable?: boolean;
  isMonospace?: boolean;

  // Loading state
  skeleton?: () => ReactNode;
}

/**
 * Extended column definition with custom metadata
 */
export type DataTableColumnDef<TData> = ColumnDef<TData, unknown> & {
  meta?: DataTableColumnMeta;
};

// Extend TanStack Table's module to include custom meta
declare module "@tanstack/react-table" {
  // biome-ignore lint/correctness/noUnusedVariables: Module augmentation requires these type parameters
  interface ColumnMeta<TData, TValue> extends DataTableColumnMeta {}
}

/**
 * Table layout mode
 */
export type LayoutMode = "grid" | "classic";

/**
 * Table configuration options
 */
export interface DataTableConfig {
  // Dimensions
  rowHeight: number; // Default: 36
  headerHeight: number; // Default: 40
  rowSpacing: number; // Default: 4 (classic mode)

  // Layout
  layout: LayoutMode; // Default: "classic"
  rowBorders: boolean; // Default: false
  containerPadding: string; // Default: "px-2"
  tableLayout: "fixed" | "auto"; // Default: "fixed"

  // Virtualization
  overscan: number; // Default: 5

  // Loading
  loadingRows: number; // Default: 10

  // Throttle delay for load more
  throttleDelay: number; // Default: 350
}

/**
 * Load more footer configuration
 */
export interface LoadMoreFooterProps {
  itemLabel?: string;
  buttonText?: string;
  countInfoText?: ReactNode;
  headerContent?: ReactNode;
  hasMore?: boolean;
  hide?: boolean;
}

/**
 * Main DataTable component props
 */
export interface DataTableProps<TData> {
  // Data (required)
  data: TData[];
  columns: DataTableColumnDef<TData>[];
  getRowId: (row: TData) => string;

  // Real-time data (optional)
  realtimeData?: TData[];

  // State management (optional, controlled)
  sorting?: SortingState;
  onSortingChange?: (sorting: SortingState | ((old: SortingState) => SortingState)) => void;
  rowSelection?: RowSelectionState;
  onRowSelectionChange?: (
    selection: RowSelectionState | ((old: RowSelectionState) => RowSelectionState),
  ) => void;

  // Interaction (optional)
  onRowClick?: (row: TData | null) => void;
  onRowMouseEnter?: (row: TData) => void;
  onRowMouseLeave?: () => void;
  selectedItem?: TData | null;

  // Pagination (optional)
  onLoadMore?: () => void;
  hasMore?: boolean;
  isFetchingNextPage?: boolean;

  // Configuration (optional)
  config?: Partial<DataTableConfig>;

  // UI customization (optional)
  emptyState?: ReactNode;
  loadMoreFooterProps?: LoadMoreFooterProps;
  rowClassName?: (row: TData) => string;
  selectedClassName?: (row: TData, isSelected: boolean) => string;
  fixedHeight?: number;

  // Features (optional)
  enableKeyboardNav?: boolean;
  enableSorting?: boolean;
  enableRowSelection?: boolean;

  // Loading state
  isLoading?: boolean;

  // Custom skeleton renderer
  renderSkeletonRow?: (props: {
    columns: DataTableColumnDef<TData>[];
    rowHeight: number;
  }) => ReactNode;
}

/**
 * Internal separator item type for real-time data boundary
 */
export type SeparatorItem = {
  isSeparator: true;
};

/**
 * Combined data type including separator
 */
export type TableDataItem<TData> = TData | SeparatorItem;
