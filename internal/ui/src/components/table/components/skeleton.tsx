import type { Table } from "@tanstack/react-table";
import { TABLE_CLASS_NAMES } from "../constants";

interface TableSkeletonProps<TData = unknown> {
  table: Table<TData>;
  rows?: number;
}

export function TableSkeleton<TData = unknown>({ table, rows = 10 }: TableSkeletonProps<TData>) {
  const columns = table.getVisibleLeafColumns();

  return (
    <div className={TABLE_CLASS_NAMES.skeleton}>
      {Array.from({ length: rows }, (_, i) => i).map((rowIndex) => (
        <div
          key={`skeleton-row-${rowIndex}`}
          className="flex border-b border-gray-200 dark:border-gray-800"
        >
          {columns.map((column) => (
            <div
              key={`skeleton-cell-${rowIndex}-${column.id}`}
              className="flex-1 p-4"
              style={{
                width: column.getSize(),
                minWidth: column.columnDef.minSize,
                maxWidth: column.columnDef.maxSize,
              }}
            >
              <div className="h-4 bg-gray-200 dark:bg-gray-800 rounded animate-pulse" />
            </div>
          ))}
        </div>
      ))}
    </div>
  );
}

export function SkeletonCell() {
  return <div className="h-4 bg-gray-200 dark:bg-gray-800 rounded animate-pulse w-full" />;
}

export function SkeletonRow({ columnCount = 5 }: { columnCount?: number }) {
  return (
    <div className="flex border-b border-gray-200 dark:border-gray-800 p-4 gap-4">
      {Array.from({ length: columnCount }, (_, i) => i).map((index) => (
        <div key={`skeleton-col-${index}`} className="flex-1">
          <SkeletonCell />
        </div>
      ))}
    </div>
  );
}
