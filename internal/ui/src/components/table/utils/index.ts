export * from "./export";
export * from "./storage";

/**
 * Generate a unique ID for rows
 */
export function generateRowId<TData>(
  row: TData,
  index: number,
  getRowId?: (row: TData, index: number) => string,
): string {
  if (getRowId) {
    return getRowId(row, index);
  }

  // Try to find an ID property
  if (row && typeof row === "object") {
    if ("id" in row && typeof row.id === "string") {
      return row.id;
    }
    if ("_id" in row && typeof row._id === "string") {
      return row._id;
    }
  }

  // Fallback to index
  return `row-${index}`;
}

/**
 * Safely get a value from an object by path
 */
export function getValueByPath(obj: Record<string, unknown>, path: string): unknown {
  return path.split(".").reduce((current, key) => {
    if (current && typeof current === "object" && key in current) {
      return (current as Record<string, unknown>)[key];
    }
    return undefined;
  }, obj as unknown);
}

/**
 * Check if a value matches a filter
 */
export function matchesFilter(value: unknown, filterValue: string): boolean {
  if (value == null) {
    return false;
  }

  const stringValue = String(value).toLowerCase();
  const filter = filterValue.toLowerCase();

  return stringValue.includes(filter);
}

/**
 * Format date for display
 */
export function formatDate(date: Date | string | number): string {
  if (!date) {
    return "";
  }

  try {
    const d = new Date(date);
    return `${d.toLocaleDateString()} ${d.toLocaleTimeString()}`;
  } catch {
    return String(date);
  }
}

/**
 * Format number with commas
 */
export function formatNumber(num: number): string {
  return new Intl.NumberFormat().format(num);
}

/**
 * Truncate text with ellipsis
 */
export function truncate(text: string, maxLength: number): string {
  if (text.length <= maxLength) {
    return text;
  }
  return `${text.slice(0, maxLength - 3)}...`;
}

/**
 * Merge class names
 */
export function cn(...classes: (string | undefined | null | false)[]): string {
  return classes.filter(Boolean).join(" ");
}
