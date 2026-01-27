import type { ReactNode } from "react";

type ColumnWidth =
  | number // Fixed pixel width
  | string // CSS values like "165px", "15%", etc.
  | "1fr" // Flex grow
  | "min" // Minimum width based on content
  | "auto" // Automatic width based on content
  | { min: number; max: number } // Responsive range
  | { flex: number }; // Flex grow ratio

export type SortDirection = "asc" | "desc" | null;
export type Column<TTableData> = {
  key: string;
  header?: string;
  width: ColumnWidth;
  headerClassName?: string;
  render: (item: TTableData) => React.ReactNode;
  sort?: {
    sortable?: boolean;
    onSort?: (direction: SortDirection) => void;
    direction?: SortDirection;
  };
};

export type TableLayoutMode = "classic" | "grid";

export interface TableConfig {
  rowHeight: number;
  loadingRows: number;
  overscan: number;
  throttleDelay: number;
  headerHeight: number;

  // Layout options
  layoutMode?: TableLayoutMode; // 'classic' or 'grid'
  rowBorders?: boolean; // Add borders between rows
  containerPadding?: string; // Custom padding for container (e.g., 'px-0', 'px-4', 'p-2')
  rowSpacing?: number; // Space between rows in pixels (for classic mode)
}

export type VirtualTableProps<TTableData> = {
  data: TTableData[];
  realtimeData?: TTableData[];
  columns: Column<TTableData>[];
  isLoading?: boolean;
  config?: Partial<TableConfig>;
  onRowClick?: (item: TTableData | null) => void;
  onLoadMore?: () => void;
  emptyState?: React.ReactNode;
  keyExtractor: (item: TTableData) => string | number;
  rowClassName?: (item: TTableData) => string;
  selectedClassName?: (item: TTableData, isSelected: boolean) => string;
  selectedItem?: TTableData | null;
  isFetchingNextPage?: boolean;
  loadMoreFooterProps?: {
    itemLabel?: string;
    buttonText?: string;
    countInfoText?: React.ReactNode;
    headerContent?: React.ReactNode;
    hasMore?: boolean;
    hide?: boolean;
  };
  renderSkeletonRow?: (props: {
    columns: Column<TTableData>[];
    rowHeight: number;
  }) => ReactNode;
  /**
   * Callback when mouse enters a row
   */
  onRowMouseEnter?: (item: TTableData) => void;
  /**
   * Callback when mouse leaves a row
   */
  onRowMouseLeave?: () => void;
  /**
   * Override auto-calculated table height in pixels
   */
  fixedHeight?: number;
};

export type SeparatorItem = {
  isSeparator: true;
};
