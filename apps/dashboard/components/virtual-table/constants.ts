import type { TableConfig } from "./types";

export const DEFAULT_CONFIG: TableConfig = {
  rowHeight: 26,
  loadingRows: 50,
  overscan: 5,
  throttleDelay: 350,
  headerHeight: 40,
  layoutMode: "classic", // Default to classic table layout
  rowBorders: false, // Default to no borders
  containerPadding: "px-2", // Default container padding
  rowSpacing: 4, // Default spacing between rows (classic mode)
} as const;
