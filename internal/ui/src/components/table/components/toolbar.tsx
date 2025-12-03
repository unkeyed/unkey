"use client";
import type { Table } from "@tanstack/react-table";
import React from "react";
import { A11Y_LABELS, TABLE_CLASS_NAMES } from "../constants";
import type { ExportConfig } from "../types";
import { exportTable } from "../utils";

interface TableToolbarProps<TData> {
  table: Table<TData>;
  enableGlobalFilter?: boolean;
  enableColumnVisibility?: boolean;
  enableExport?: boolean;
  exportConfig?: ExportConfig;
}

export function TableToolbar<TData>({
  table,
  enableGlobalFilter = false,
  enableColumnVisibility = false,
  enableExport = false,
  exportConfig,
}: TableToolbarProps<TData>) {
  const [globalFilter, setGlobalFilter] = React.useState("");
  const [showColumnVisibility, setShowColumnVisibility] = React.useState(false);

  const handleExport = (format: "csv" | "json") => {
    exportTable(table, {
      format,
      filename: exportConfig?.defaultFilename,
      includeHeaders: true,
      selectedRowsOnly: false,
      visibleColumnsOnly: true,
    });
  };

  if (!enableGlobalFilter && !enableColumnVisibility && !enableExport) {
    return null;
  }

  return (
    <div
      className={`${TABLE_CLASS_NAMES.toolbar} flex items-center justify-between gap-4 p-4 border-b border-gray-200 dark:border-gray-800`}
    >
      {/* Left side - Global filter */}
      <div className="flex-1">
        {enableGlobalFilter && (
          <input
            type="text"
            placeholder={A11Y_LABELS.globalSearch}
            value={globalFilter}
            onChange={(e) => {
              setGlobalFilter(e.target.value);
              table.setGlobalFilter(e.target.value);
            }}
            className="w-full max-w-sm px-3 py-2 border border-gray-300 dark:border-gray-700 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500 dark:bg-gray-900 dark:text-gray-100"
            aria-label={A11Y_LABELS.globalSearch}
          />
        )}
      </div>

      {/* Right side - Actions */}
      <div className="flex items-center gap-2">
        {/* Column Visibility */}
        {enableColumnVisibility && (
          <div className="relative">
            <button
              type="button"
              onClick={() => setShowColumnVisibility(!showColumnVisibility)}
              className="px-3 py-2 border border-gray-300 dark:border-gray-700 rounded-md shadow-sm text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-900 hover:bg-gray-50 dark:hover:bg-gray-800 focus:outline-none focus:ring-2 focus:ring-primary-500"
              aria-label={A11Y_LABELS.columnVisibility}
            >
              Columns
            </button>

            {showColumnVisibility && (
              <div className="absolute right-0 mt-2 w-56 rounded-md shadow-lg bg-white dark:bg-gray-900 ring-1 ring-black ring-opacity-5 z-10">
                <div className="py-1 max-h-64 overflow-auto">
                  {table.getAllLeafColumns().map((column) => {
                    const header =
                      typeof column.columnDef.header === "string"
                        ? column.columnDef.header
                        : column.id;

                    return (
                      <label
                        key={column.id}
                        className="flex items-center px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 cursor-pointer"
                      >
                        <input
                          type="checkbox"
                          checked={column.getIsVisible()}
                          onChange={column.getToggleVisibilityHandler()}
                          className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-gray-300 rounded"
                        />
                        <span className="ml-2">{header}</span>
                      </label>
                    );
                  })}
                </div>
              </div>
            )}
          </div>
        )}

        {/* Export */}
        {enableExport && (
          <>
            {exportConfig?.enableCSV !== false && (
              <button
                type="button"
                onClick={() => handleExport("csv")}
                className="px-3 py-2 border border-gray-300 dark:border-gray-700 rounded-md shadow-sm text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-900 hover:bg-gray-50 dark:hover:bg-gray-800 focus:outline-none focus:ring-2 focus:ring-primary-500"
                aria-label="Export as CSV"
              >
                Export CSV
              </button>
            )}
            {exportConfig?.enableJSON !== false && (
              <button
                type="button"
                onClick={() => handleExport("json")}
                className="px-3 py-2 border border-gray-300 dark:border-gray-700 rounded-md shadow-sm text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-900 hover:bg-gray-50 dark:hover:bg-gray-800 focus:outline-none focus:ring-2 focus:ring-primary-500"
                aria-label="Export as JSON"
              >
                Export JSON
              </button>
            )}
          </>
        )}
      </div>
    </div>
  );
}
