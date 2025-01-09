export type Column<T> = {
  key: string;
  header: string;
  headerClassName?: string;
  width: string;
  render: (item: T) => React.ReactNode;
};

export type TableConfig = {
  rowHeight: number;
  loadingRows: number;
  overscan: number;
  tableBorder: number;
  throttleDelay: number;
  headerHeight: number;
};

export type VirtualTableProps<T> = {
  data: T[];
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
  renderDetails?: (item: T, onClose: () => void, distanceToTop: number) => React.ReactNode;
};
