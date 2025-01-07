import { Column } from "../types";

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
      <div key={column.key} className="px-2">
        <div className="h-4 bg-accent-6 rounded" />
      </div>
    ))}
  </div>
);
