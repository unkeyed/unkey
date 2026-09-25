import type { DataTableConfig } from "../types";

export const DEFAULT_CONFIG: DataTableConfig = {
  rowHeight: 36,
  headerHeight: 40,
  rowSpacing: 4,

  layout: "classic",
  rowBorders: false,
  containerPadding: "px-2",
  tableLayout: "fixed",

  loadingRows: 10,
} as const;

export const MOBILE_TABLE_HEIGHT = 400;

export const BREATHING_SPACE = 10;

export type StatusStyle = {
  base: string;
  hover: string;
  selected: string;
  badge: { default: string; selected: string };
  focusRing: string;
};

export const FOOTER_PANEL =
  "w-[740px] bg-raised min-h-[60px] flex items-center justify-center rounded-xl shadow-floating mb-5";

export const STATUS_STYLES: StatusStyle = {
  base: "text-grayA-9",
  hover: "hover:text-gray-11 dark:hover:text-gray-12 hover:bg-grayA-2",
  selected: "text-gray-12 bg-grayA-2 hover:text-gray-12",
  badge: {
    default: "bg-grayA-3 text-grayA-11 group-hover:bg-grayA-5 border-transparent",
    selected: "bg-grayA-5 text-grayA-12 hover:bg-grayA-5 border-grayA-3",
  },
  focusRing: "focus:ring-gray-7",
};
