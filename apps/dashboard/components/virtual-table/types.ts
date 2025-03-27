type ColumnWidth =
  | number // Fixed pixel width
  | string // CSS values like "165px", "15%", etc.
  | "1fr" // Flex grow
  | "min" // Minimum width based on content
  | "auto" // Automatic width based on content
  | { min: number; max: number } // Responsive range
  | { flex: number }; // Flex grow ratio

export type SortDirection = "asc" | "desc" | null;
export type Column<T> = {
  key: string;
  header?: string;
  width: ColumnWidth;
  headerClassName?: string;
  render: (item: T) => React.ReactNode;
  sort?: {
    sortable?: boolean;
    onSort?: (direction: SortDirection) => void;
    direction?: SortDirection;
  };
};

export type TableConfig = {
  rowHeight: number;
  loadingRows: number;
  overscan: number;
  throttleDelay: number;
  headerHeight: number;
};

export type VirtualTableProps<T> = {
  data: T[];
  realtimeData?: T[];
  columns: Column<T>[];
  isLoading?: boolean;
  config?: Partial<TableConfig>;
  onRowClick?: (item: T | null) => void;
  onLoadMore?: () => void;
  emptyState?: React.ReactNode;
  keyExtractor: (item: T) => string | number;
  rowClassName?: (item: T) => string;
  focusClassName?: (item: T) => string;
  selectedClassName?: (item: T, isSelected: boolean) => string;
  selectedItem?: T | null;
  isFetchingNextPage?: boolean;
};

export type SeparatorItem = {
  isSeparator: true;
};
