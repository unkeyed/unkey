import type {
  ColumnResizeConfig,
  EmptyStateConfig,
  ExportConfig,
  KeyboardConfig,
  LoadingConfig,
  PaginationConfig,
  RealtimeConfig,
  SelectionConfig,
  SortConfig,
} from "./types";

// ============================================================================
// Default Table Configuration
// ============================================================================

export const DEFAULT_ROW_HEIGHT = 52;
export const DEFAULT_SKELETON_ROWS = 10;
export const DEFAULT_OVERSCAN = 5;
export const DEFAULT_PAGE_SIZE = 50;
export const DEFAULT_PAGE_SIZE_OPTIONS = [10, 25, 50, 100, 200];

// ============================================================================
// Default Pagination Config
// ============================================================================

export const DEFAULT_PAGINATION_CONFIG: PaginationConfig = {
  mode: "client",
  pageSize: DEFAULT_PAGE_SIZE,
  pageSizeOptions: DEFAULT_PAGE_SIZE_OPTIONS,
  showPageSizeSelector: true,
  showPageInfo: true,
  showFirstLast: true,
};

// ============================================================================
// Default Sort Config
// ============================================================================

export const DEFAULT_SORT_CONFIG: SortConfig = {
  enableMultiSort: true,
  maxMultiSortColCount: 3,
  sortDescFirst: false,
  isMultiSortEvent: (e: unknown) => {
    const event = e as KeyboardEvent;
    return event?.shiftKey === true;
  },
};

// ============================================================================
// Default Selection Config
// ============================================================================

export const DEFAULT_SELECTION_CONFIG: SelectionConfig = {
  enableMultiRowSelection: true,
  enableSubRowSelection: false,
  enableSelectAll: true,
  selectAllMode: "page",
};

// ============================================================================
// Default Keyboard Config
// ============================================================================

export const DEFAULT_KEYBOARD_CONFIG: KeyboardConfig = {
  enableVimBindings: true,
  customKeyMap: {
    ArrowUp: "moveUp",
    ArrowDown: "moveDown",
    ArrowLeft: "moveLeft",
    ArrowRight: "moveRight",
    j: "moveDown",
    k: "moveUp",
    Enter: "selectRow",
    " ": "toggleSelection",
    Escape: "escape",
  },
};

// ============================================================================
// Default Realtime Config
// ============================================================================

export const DEFAULT_REALTIME_CONFIG: RealtimeConfig = {
  enabled: false,
  pollingInterval: 5000,
  separatorLabel: "Live",
  highlightNew: true,
  highlightDuration: 3000,
  maxRealtimeItems: 100,
};

// ============================================================================
// Default Column Resize Config
// ============================================================================

export const DEFAULT_COLUMN_RESIZE_CONFIG: ColumnResizeConfig = {
  enabled: true,
  mode: "onChange",
  minWidth: 50,
  maxWidth: 1000,
  defaultWidth: 150,
};

// ============================================================================
// Default Export Config
// ============================================================================

export const DEFAULT_EXPORT_CONFIG: ExportConfig = {
  enableCSV: true,
  enableJSON: true,
  enableXLSX: false,
  defaultFilename: "table-export",
};

// ============================================================================
// Default Loading Config
// ============================================================================

export const DEFAULT_LOADING_CONFIG: LoadingConfig = {
  showSkeleton: true,
  skeletonRows: DEFAULT_SKELETON_ROWS,
  loadingOverlay: false,
  loadingMessage: "Loading...",
};

// ============================================================================
// Default Empty State Config
// ============================================================================

export const DEFAULT_EMPTY_STATE_CONFIG: EmptyStateConfig = {
  title: "No data available",
  description: "There are no items to display",
};

// ============================================================================
// Keyboard Key Maps
// ============================================================================

export const KEYBOARD_KEYS = {
  ARROW_UP: "ArrowUp",
  ARROW_DOWN: "ArrowDown",
  ARROW_LEFT: "ArrowLeft",
  ARROW_RIGHT: "ArrowRight",
  ENTER: "Enter",
  SPACE: " ",
  ESCAPE: "Escape",
  TAB: "Tab",
  HOME: "Home",
  END: "End",
  PAGE_UP: "PageUp",
  PAGE_DOWN: "PageDown",
  VIM_DOWN: "j",
  VIM_UP: "k",
  VIM_LEFT: "h",
  VIM_RIGHT: "l",
} as const;

// ============================================================================
// CSS Class Names
// ============================================================================

export const TABLE_CLASS_NAMES = {
  container: "unkey-table-container",
  wrapper: "unkey-table-wrapper",
  table: "unkey-table",
  header: "unkey-table-header",
  headerRow: "unkey-table-header-row",
  headerCell: "unkey-table-header-cell",
  body: "unkey-table-body",
  row: "unkey-table-row",
  cell: "unkey-table-cell",
  footer: "unkey-table-footer",
  toolbar: "unkey-table-toolbar",
  empty: "unkey-table-empty",
  loading: "unkey-table-loading",
  skeleton: "unkey-table-skeleton",
  virtualPadding: "unkey-table-virtual-padding",
  realtimeSeparator: "unkey-table-realtime-separator",
  selected: "unkey-table-row-selected",
  hoverable: "unkey-table-row-hoverable",
  clickable: "unkey-table-row-clickable",
  resizer: "unkey-table-column-resizer",
  sortable: "unkey-table-header-sortable",
  sorted: "unkey-table-header-sorted",
  filtered: "unkey-table-header-filtered",
} as const;

// ============================================================================
// Accessibility Labels
// ============================================================================

export const A11Y_LABELS = {
  table: "Data table",
  sortAscending: "Sorted ascending",
  sortDescending: "Sorted descending",
  sortNone: "Not sorted",
  filter: "Filter column",
  globalSearch: "Search all columns",
  selectRow: "Select row",
  selectAll: "Select all rows",
  deselectAll: "Deselect all rows",
  columnVisibility: "Toggle column visibility",
  export: "Export data",
  loadMore: "Load more items",
  loading: "Loading data",
  noData: "No data available",
  pagination: {
    first: "Go to first page",
    previous: "Go to previous page",
    next: "Go to next page",
    last: "Go to last page",
    pageInfo: "Page {current} of {total}",
    pageSize: "Items per page",
  },
} as const;

// ============================================================================
// Storage Keys
// ============================================================================

export const STORAGE_KEYS = {
  columnOrder: "column-order",
  columnSizing: "column-sizing",
  columnVisibility: "column-visibility",
  pageSize: "page-size",
} as const;

// ============================================================================
// Export Formats
// ============================================================================

export const EXPORT_FORMATS = {
  CSV: {
    extension: ".csv",
    mimeType: "text/csv;charset=utf-8;",
  },
  JSON: {
    extension: ".json",
    mimeType: "application/json",
  },
  XLSX: {
    extension: ".xlsx",
    mimeType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
  },
} as const;

// ============================================================================
// Debounce Delays
// ============================================================================

export const DEBOUNCE_DELAYS = {
  FILTER: 300,
  SEARCH: 300,
  RESIZE: 150,
  STORAGE: 500,
} as const;

// ============================================================================
// Animation Durations
// ============================================================================

export const ANIMATION_DURATIONS = {
  HIGHLIGHT: 3000,
  SKELETON_PULSE: 2000,
  ROW_FADE: 200,
} as const;
