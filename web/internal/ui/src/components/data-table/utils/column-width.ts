import type { ColumnWidth } from "../types";

/**
 * Convert ColumnWidth to CSS width string
 */
export const calculateColumnWidth = (width?: ColumnWidth): string => {
  if (!width) {
    return "auto";
  }

  if (typeof width === "number") {
    return `${width}px`;
  }

  if (typeof width === "string") {
    return width;
  }

  if (typeof width === "object") {
    if ("min" in width && "max" in width) {
      return `${width.min}px`;
    }
    if ("flex" in width) {
      return "auto";
    }
  }

  return "auto";
};
