"use client";
// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import * as React from "react";

import { type Row, type Table, flexRender } from "@tanstack/react-table";
import type { VirtualItem } from "@tanstack/react-virtual";
import { TABLE_CLASS_NAMES } from "../constants";
import { EmptyState } from "./empty-state";
import { TableSkeleton } from "./skeleton";

interface TableBodyProps<TData> {
  table: Table<TData>;
  virtualRows?: VirtualItem[];
  paddingTop?: number;
  paddingBottom?: number;
  isLoading?: boolean;
  emptyState?: React.ReactNode;
  loadingState?: React.ReactNode;
  onRowClick?: (row: TData) => void;
  rowClassName?: string | ((row: TData) => string);
  focusedRowIndex?: number;
}

export function TableBody<TData>({
  table,
  virtualRows,
  paddingTop = 0,
  paddingBottom = 0,
  isLoading = false,
  emptyState,
  loadingState,
  onRowClick,
  rowClassName,
  focusedRowIndex = -1,
}: TableBodyProps<TData>) {
  const rows = table.getRowModel().rows;

  // Show loading state
  if (isLoading && rows.length === 0) {
    if (loadingState) {
      return (
        <tbody className={TABLE_CLASS_NAMES.body}>
          <tr>
            <td colSpan={table.getAllColumns().length}>{loadingState}</td>
          </tr>
        </tbody>
      );
    }
    return (
      <tbody className={TABLE_CLASS_NAMES.body}>
        <tr>
          <td colSpan={table.getAllColumns().length} className="p-0">
            <TableSkeleton table={table} />
          </td>
        </tr>
      </tbody>
    );
  }

  // Show empty state
  if (!isLoading && rows.length === 0) {
    return (
      <tbody className={TABLE_CLASS_NAMES.body}>
        <tr>
          <td colSpan={table.getAllColumns().length}>{emptyState || <EmptyState />}</td>
        </tr>
      </tbody>
    );
  }

  // Determine which rows to render (virtual or all)
  const rowsToRender = virtualRows ? virtualRows.map((virtualRow) => rows[virtualRow.index]) : rows;

  return (
    <tbody className={TABLE_CLASS_NAMES.body}>
      {/* Top padding for virtualization */}
      {paddingTop > 0 && (
        <tr>
          <td
            colSpan={table.getAllColumns().length}
            style={{ height: `${paddingTop}px` }}
            className={TABLE_CLASS_NAMES.virtualPadding}
          />
        </tr>
      )}

      {/* Render rows */}
      {rowsToRender.map((row: Row<TData> | undefined, index: number) => {
        if (!row) {
          return null;
        }

        const rowIndex = virtualRows ? virtualRows[index]?.index : index;
        const isFocused = rowIndex === focusedRowIndex;
        const isSelected = row.getIsSelected();

        const computedRowClassName =
          typeof rowClassName === "function" ? rowClassName(row.original) : rowClassName;

        return (
          <tr
            key={row.id}
            data-row-index={rowIndex}
            className={`${TABLE_CLASS_NAMES.row} ${computedRowClassName || ""} ${onRowClick ? `${TABLE_CLASS_NAMES.clickable} cursor-pointer` : ""} ${isFocused ? "bg-blue-50 dark:bg-blue-900/20" : ""} ${isSelected ? `${TABLE_CLASS_NAMES.selected} bg-gray-100 dark:bg-gray-800` : ""} ${!isFocused && !isSelected ? "hover:bg-gray-50 dark:hover:bg-gray-900" : ""} border-b border-gray-200 dark:border-gray-800`}
            onClick={() => onRowClick?.(row.original)}
            onKeyDown={
              onRowClick
                ? (e) => {
                    if (e.key === "Enter" || e.key === " ") {
                      e.preventDefault();
                      onRowClick(row.original);
                    }
                  }
                : undefined
            }
            tabIndex={onRowClick ? 0 : undefined}
            role={onRowClick ? "button" : undefined}
          >
            {row.getVisibleCells().map((cell) => (
              <td
                key={cell.id}
                className={`${TABLE_CLASS_NAMES.cell} px-4 py-3 text-sm text-gray-900 dark:text-gray-100`}
                style={{
                  width: cell.column.getSize(),
                  minWidth: cell.column.columnDef.minSize,
                  maxWidth: cell.column.columnDef.maxSize,
                }}
              >
                {flexRender(cell.column.columnDef.cell, cell.getContext())}
              </td>
            ))}
          </tr>
        );
      })}

      {/* Bottom padding for virtualization */}
      {paddingBottom > 0 && (
        <tr>
          <td
            colSpan={table.getAllColumns().length}
            style={{ height: `${paddingBottom}px` }}
            className={TABLE_CLASS_NAMES.virtualPadding}
          />
        </tr>
      )}
    </tbody>
  );
}
