import type { Column } from "../types";

export const LoadingRow = <TTableItem,>({
  columns,
}: {
  columns: Column<TTableItem>[];
}) => (
  <div
    className="grid text-sm animate-pulse mt-1"
    style={{ gridTemplateColumns: columns.map((col) => col.width).join(" ") }}
  >
    {columns.map((column) => (
      <div key={column.key} className="pr-1">
        <div className="h-4 bg-gradient-to-r from-accent-6 to-accent-4 rounded" />
      </div>
    ))}
  </div>
);
