import type { DataTableConfig } from "../types";

export const DEFAULT_CONFIG: DataTableConfig = {
  rowHeight: 36,
  headerHeight: 40,
  rowSpacing: 4,

  layout: "classic",
  rowBorders: false,
  containerPadding: "px-2",
  tableLayout: "fixed",
  striped: false,

  loadingRows: 10,
} as const;

export type StatusStyle = {
  base: string;
  hover: string;
  selected: string;
  badge: { default: string; selected: string };
  focusRing: string;
};

export const STATUS_STYLES: StatusStyle = {
  base: "text-gray-12",
  hover: "hover:text-gray-12 hover:bg-grayA-3",
  selected: "text-gray-12 bg-grayA-3 hover:text-gray-12",
  badge: {
    default: "bg-grayA-3 text-gray-12 group-hover:bg-grayA-5 border-transparent",
    selected: "bg-grayA-5 text-grayA-12 hover:bg-grayA-5 border-grayA-3",
  },
  focusRing: "focus:ring-gray-7",
};
