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

  // Loading
  loadingRows: 10,
} as const;

/**
 * Mobile table height constant
 */
export const MOBILE_TABLE_HEIGHT = 400;

/**
 * Breathing space for table height calculation
 */
export const BREATHING_SPACE = 10;

export type StatusStyle = {
  base: string;
  hover: string;
  selected: string;
  badge: { default: string; selected: string };
  focusRing: string;
};

export const STATUS_STYLES: StatusStyle = {
  base: "text-grayA-9",
  hover: "hover:text-accent-11 dark:hover:text-accent-12 hover:bg-grayA-2",
  selected: "text-accent-12 bg-grayA-2 hover:text-accent-12",
  badge: {
    default: "bg-grayA-3 text-grayA-11 group-hover:bg-grayA-5 border-transparent",
    selected: "bg-grayA-5 text-grayA-12 hover:bg-grayA-5 border-grayA-3",
  },
  focusRing: "focus:ring-accent-7",
};
