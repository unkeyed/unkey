import type { TableConfig } from "./types";

export const DEFAULT_CONFIG: TableConfig = {
  rowHeight: 26,
  loadingRows: 50,
  overscan: 5,
  throttleDelay: 350,
  headerHeight: 40,
} as const;
