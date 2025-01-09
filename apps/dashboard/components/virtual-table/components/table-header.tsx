import type { Column } from "../types";

export const TableHeader = <TTableItem,>({
  columns,
}: {
  columns: Column<TTableItem>[];
}) => (
  <>
    <div
      className="grid text-sm font-medium text-accent-12 py-1"
      style={{
        gridTemplateColumns: columns.map((col) => col.width).join(" "),
      }}
    >
      {columns.map((column) => (
        <div key={column.key} className={column.headerClassName}>
          <div className="truncate text-accent-12">{column.header}</div>
        </div>
      ))}
    </div>
    <div className="w-full border-t border-border" />
  </>
);
