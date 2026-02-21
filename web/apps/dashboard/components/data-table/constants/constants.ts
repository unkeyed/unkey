import type { DataTableConfig } from "../types";

/**
 * Default configuration for DataTable
 */
export const DEFAULT_CONFIG: DataTableConfig = {
  // Dimensions
  rowHeight: 36,
  headerHeight: 40,
  rowSpacing: 4,

  // Layout
  layout: "classic",
  rowBorders: false,
  containerPadding: "px-2",
  tableLayout: "fixed",

  // Virtualization
  overscan: 5,

  // Loading
  loadingRows: 10,

  // Throttle
  throttleDelay: 350,
} as const;

/**
 * Mobile table height constant
 */
export const MOBILE_TABLE_HEIGHT = 400;

/**
 * Breathing space for table height calculation
 */
export const BREATHING_SPACE = 20;
