"use client";
import {
  type ColumnDef,
  type ColumnFiltersState,
  type RowSelectionState,
  type SortingState,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from "@tanstack/react-table";
import React from "react";
import { TableBody } from "./components/table-body";
import { TableHeader } from "./components/table-header";
import { TableToolbar } from "./components/toolbar";
import { TABLE_CLASS_NAMES } from "./constants";
import { useKeyboardNavigation, useTableState, useTableVirtualizer } from "./hooks";
import type { TableProps } from "./types";

export function Table<TData>(props: TableProps<TData>) {
  const {
    data,
    columns,
    // State
    sorting: externalSorting,
    onSortingChange,
    columnFilters: externalColumnFilters,
    onColumnFiltersChange,
    globalFilter: externalGlobalFilter,
    onGlobalFilterChange,
    pagination: externalPagination,
    onPaginationChange,
    rowSelection: externalRowSelection,
    onRowSelectionChange,
    columnVisibility: externalColumnVisibility,
    onColumnVisibilityChange,
    columnOrder: externalColumnOrder,
    onColumnOrderChange,
    columnSizing: externalColumnSizing,
    onColumnSizingChange,
    // Features
    enableSorting = true,
    enableFiltering = false,
    enableGlobalFilter = false,
    enablePagination = false,
    enableRowSelection = false,
    enableMultiRowSelection = true,
    enableColumnResizing = false,
    enableColumnVisibility = false,
    enableVirtualization = true,
    enableInfiniteScroll = false,
    enableExport = false,
    enableKeyboardNavigation = true,
    // Data mode
    manualSorting = false,
    manualFiltering = false,
    manualPagination = false,
    // Data fetching
    onFetchMore,
    hasMore = false,
    isLoading = false,
    isLoadingMore = false,
    totalCount,
    // Virtualization
    estimateSize = 52,
    overscan = 5,
    // Layout
    stickyHeader = false,
    maxHeight = "600px",
    className = "",
    rowClassName,
    // States
    emptyState,
    loadingState,
    // Callbacks
    onRowClick,
    getRowId,
    // Persistence
    persistenceKey,
    persistColumnOrder = false,
    persistColumnSizing = false,
    persistColumnVisibility = false,
    // Export
    exportFilename = "table-export",
    exportFormats,
    // Accessibility
    ariaLabel,
  } = props;

  // Internal state management with persistence
  const tableState = useTableState({
    persistenceKey,
    persistColumnOrder,
    persistColumnSizing,
    persistColumnVisibility,
  });

  // State handlers
  const [internalSorting, setInternalSorting] = React.useState<SortingState>([]);
  const [internalColumnFilters, setInternalColumnFilters] = React.useState<ColumnFiltersState>([]);
  const [internalGlobalFilter, setInternalGlobalFilter] = React.useState<string>("");
  const [internalPagination, setInternalPagination] = React.useState({
    pageIndex: 0,
    pageSize: 50,
  });
  const [internalRowSelection, setInternalRowSelection] = React.useState<RowSelectionState>({});

  // Use external state if provided, otherwise use internal
  const sorting = externalSorting ?? internalSorting;
  const setSorting = onSortingChange ?? setInternalSorting;
  const columnFilters = externalColumnFilters ?? internalColumnFilters;
  const setColumnFilters = onColumnFiltersChange ?? setInternalColumnFilters;
  const globalFilter = externalGlobalFilter ?? internalGlobalFilter;
  const setGlobalFilter = onGlobalFilterChange ?? setInternalGlobalFilter;
  const pagination = externalPagination ?? internalPagination;
  const setPagination = onPaginationChange ?? setInternalPagination;
  const rowSelection = externalRowSelection ?? internalRowSelection;
  const setRowSelection = onRowSelectionChange ?? setInternalRowSelection;
  const columnVisibility = externalColumnVisibility ?? tableState.columnVisibility;
  const setColumnVisibility = onColumnVisibilityChange ?? tableState.setColumnVisibility;
  const columnOrder = externalColumnOrder ?? tableState.columnOrder;
  const setColumnOrder = onColumnOrderChange ?? tableState.setColumnOrder;
  const columnSizing = externalColumnSizing ?? tableState.columnSizing;
  const setColumnSizing = onColumnSizingChange ?? tableState.setColumnSizing;

  // Create table instance
  const table = useReactTable({
    data,
    columns,
    state: {
      sorting,
      columnFilters,
      globalFilter,
      pagination,
      rowSelection,
      columnVisibility,
      columnOrder,
      columnSizing,
    },
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    onGlobalFilterChange: setGlobalFilter,
    onPaginationChange: setPagination,
    onRowSelectionChange: setRowSelection,
    onColumnVisibilityChange: setColumnVisibility,
    onColumnOrderChange: setColumnOrder,
    onColumnSizingChange: setColumnSizing,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: enableSorting && !manualSorting ? getSortedRowModel() : undefined,
    getFilteredRowModel:
      (enableFiltering || enableGlobalFilter) && !manualFiltering
        ? getFilteredRowModel()
        : undefined,
    getPaginationRowModel:
      enablePagination && !manualPagination ? getPaginationRowModel() : undefined,
    enableSorting,
    enableFilters: enableFiltering,
    enableGlobalFilter,
    enableRowSelection,
    enableMultiRowSelection,
    enableColumnResizing,
    columnResizeMode: "onChange",
    manualSorting,
    manualFiltering,
    manualPagination,
    getRowId,
  });

  // Virtualization
  const { parentRef, virtualRows, paddingTop, paddingBottom } = useTableVirtualizer({
    table,
    enabled: enableVirtualization,
    estimateSize,
    overscan,
    onLoadMore: enableInfiniteScroll ? onFetchMore : undefined,
    hasMore,
  });

  // Keyboard navigation
  const { containerRef, focusedRowIndex } = useKeyboardNavigation({
    table,
    enabled: enableKeyboardNavigation,
    onRowClick,
  });

  return (
    <div
      ref={containerRef}
      className={`${TABLE_CLASS_NAMES.container} ${className}`}
      tabIndex={enableKeyboardNavigation ? 0 : undefined}
      aria-label={ariaLabel || "Data table"}
    >
      {/* Toolbar */}
      <TableToolbar
        table={table}
        enableGlobalFilter={enableGlobalFilter}
        enableColumnVisibility={enableColumnVisibility}
        enableExport={enableExport}
        exportConfig={{
          defaultFilename: exportFilename,
          enableCSV: exportFormats?.includes("csv") ?? true,
          enableJSON: exportFormats?.includes("json") ?? true,
          enableXLSX: exportFormats?.includes("xlsx") ?? false,
        }}
      />

      {/* Table */}
      <div
        ref={parentRef}
        className={`${TABLE_CLASS_NAMES.wrapper} overflow-auto`}
        style={{ maxHeight }}
      >
        <table className={`${TABLE_CLASS_NAMES.table} w-full border-collapse`}>
          <TableHeader
            table={table}
            stickyHeader={stickyHeader}
            enableColumnResizing={enableColumnResizing}
          />
          <TableBody
            table={table}
            virtualRows={enableVirtualization ? virtualRows : undefined}
            paddingTop={enableVirtualization ? paddingTop : undefined}
            paddingBottom={enableVirtualization ? paddingBottom : undefined}
            isLoading={isLoading}
            emptyState={emptyState}
            loadingState={loadingState}
            onRowClick={onRowClick}
            rowClassName={rowClassName}
            focusedRowIndex={focusedRowIndex}
          />
        </table>
      </div>

      {/* Footer for infinite scroll */}
      {enableInfiniteScroll && hasMore && (
        <div className="flex items-center justify-center p-4 border-t border-gray-200 dark:border-gray-800">
          {isLoadingMore ? (
            <div className="text-sm text-gray-500 dark:text-gray-400">Loading more...</div>
          ) : (
            <button
              type="button"
              onClick={onFetchMore}
              className="text-sm text-primary-600 dark:text-primary-400 hover:text-primary-700 dark:hover:text-primary-300"
            >
              Load more
            </button>
          )}
        </div>
      )}

      {/* Pagination footer */}
      {enablePagination && !enableInfiniteScroll && (
        <div className="flex items-center justify-between px-4 py-3 border-t border-gray-200 dark:border-gray-800">
          <div className="text-sm text-gray-700 dark:text-gray-300">
            Showing {table.getRowModel().rows.length} of {totalCount ?? data.length} rows
          </div>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => table.setPageIndex(0)}
              disabled={!table.getCanPreviousPage()}
              className="px-3 py-1 text-sm border border-gray-300 dark:border-gray-700 rounded disabled:opacity-50 disabled:cursor-not-allowed"
            >
              First
            </button>
            <button
              type="button"
              onClick={() => table.previousPage()}
              disabled={!table.getCanPreviousPage()}
              className="px-3 py-1 text-sm border border-gray-300 dark:border-gray-700 rounded disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Previous
            </button>
            <span className="text-sm text-gray-700 dark:text-gray-300">
              Page {table.getState().pagination.pageIndex + 1} of {table.getPageCount()}
            </span>
            <button
              type="button"
              onClick={() => table.nextPage()}
              disabled={!table.getCanNextPage()}
              className="px-3 py-1 text-sm border border-gray-300 dark:border-gray-700 rounded disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Next
            </button>
            <button
              type="button"
              onClick={() => table.setPageIndex(table.getPageCount() - 1)}
              disabled={!table.getCanNextPage()}
              className="px-3 py-1 text-sm border border-gray-300 dark:border-gray-700 rounded disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Last
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

// Re-export types for convenience
export type { ColumnDef };
export type { TableProps } from "./types";
export * from "./types";
export * from "./utils";
export * from "./components/empty-state";
export * from "./components/skeleton";
export * from "./components/toolbar";
export * from "./components/table-header";
export * from "./components/table-body";
