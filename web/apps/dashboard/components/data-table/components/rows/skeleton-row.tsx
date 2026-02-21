import { cn } from "@/lib/utils";
import type { DataTableColumnDef } from "../../types";

interface SkeletonRowProps<TData> {
  columns: DataTableColumnDef<TData>[];
  rowHeight: number;
  className?: string;
}

/**
 * Skeleton row component for loading states
 * Supports custom skeleton renderers per column
 */
export function SkeletonRow<TData>({ columns, rowHeight, className }: SkeletonRowProps<TData>) {
  return (
    <>
      {columns.map((column) => (
        <td key={column.id} className={cn("pr-4", column.meta?.cellClassName, className)}>
          {column.meta?.skeleton ? (
            column.meta.skeleton()
          ) : (
            <div
              className="bg-accent-3 rounded animate-pulse"
              style={{ height: `${Math.min(rowHeight * 0.5, 16)}px` }}
            />
          )}
        </td>
      ))}
    </>
  );
}
