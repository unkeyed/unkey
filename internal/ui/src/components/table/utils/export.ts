import type { Table } from "@tanstack/react-table";
import { EXPORT_FORMATS } from "../constants";
import type { ExportOptions } from "../types";

/**
 * Export table data to CSV format
 */
export function exportToCSV<TData>(table: Table<TData>, options: ExportOptions): void {
  const {
    filename = "export",
    includeHeaders = true,
    selectedRowsOnly = false,
    visibleColumnsOnly = true,
  } = options;

  // Get rows to export
  const rows = selectedRowsOnly ? table.getSelectedRowModel().rows : table.getRowModel().rows;

  // Get columns to export
  const columns = visibleColumnsOnly ? table.getVisibleLeafColumns() : table.getAllLeafColumns();

  // Build CSV content
  const csvContent: string[] = [];

  // Add headers
  if (includeHeaders) {
    const headers = columns
      .map((col) => {
        const header = col.columnDef.header;
        if (typeof header === "string") {
          return escapeCSVValue(header);
        }
        return escapeCSVValue(col.id);
      })
      .join(",");
    csvContent.push(headers);
  }

  // Add data rows
  for (const row of rows) {
    const values = columns
      .map((col) => {
        const cell = row.getValue(col.id);
        return escapeCSVValue(String(cell ?? ""));
      })
      .join(",");
    csvContent.push(values);
  }

  // Create and download file
  const csv = csvContent.join("\n");
  downloadFile(csv, `${filename}${EXPORT_FORMATS.CSV.extension}`, EXPORT_FORMATS.CSV.mimeType);
}

/**
 * Export table data to JSON format
 */
export function exportToJSON<TData>(table: Table<TData>, options: ExportOptions): void {
  const { filename = "export", selectedRowsOnly = false, visibleColumnsOnly = true } = options;

  // Get rows to export
  const rows = selectedRowsOnly ? table.getSelectedRowModel().rows : table.getRowModel().rows;

  // Get columns to export
  const columns = visibleColumnsOnly ? table.getVisibleLeafColumns() : table.getAllLeafColumns();

  // Build JSON content
  const jsonData = rows.map((row) => {
    const rowData: Record<string, unknown> = {};
    for (const col of columns) {
      const header = typeof col.columnDef.header === "string" ? col.columnDef.header : col.id;
      rowData[header] = row.getValue(col.id);
    }
    return rowData;
  });

  // Create and download file
  const json = JSON.stringify(jsonData, null, 2);
  downloadFile(json, `${filename}${EXPORT_FORMATS.JSON.extension}`, EXPORT_FORMATS.JSON.mimeType);
}

/**
 * Generic export function that routes to specific format handlers
 */
export function exportTable<TData>(table: Table<TData>, options: ExportOptions): void {
  switch (options.format) {
    case "csv":
      exportToCSV(table, options);
      break;
    case "json":
      exportToJSON(table, options);
      break;
    case "xlsx":
      // XLSX export would require a library like xlsx or exceljs
      console.warn("XLSX export not yet implemented");
      break;
    default:
      console.error(`Unsupported export format: ${options.format}`);
  }
}

/**
 * Escape special characters in CSV values
 */
function escapeCSVValue(value: string): string {
  // If value contains comma, quotes, or newlines, wrap in quotes and escape quotes
  if (value.includes(",") || value.includes('"') || value.includes("\n")) {
    return `"${value.replace(/"/g, '""')}"`;
  }
  return value;
}

/**
 * Download a file with the given content
 */
function downloadFile(content: string, filename: string, mimeType: string): void {
  const blob = new Blob([content], { type: mimeType });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}
