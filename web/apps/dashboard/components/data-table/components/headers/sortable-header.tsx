import { cn } from "@/lib/utils";
import type { Header } from "@tanstack/react-table";
import { flexRender } from "@tanstack/react-table";
import { CaretDown, CaretExpandY, CaretUp } from "@unkey/icons";

interface SortableHeaderProps<TData> {
  header: Header<TData, unknown>;
  children?: React.ReactNode;
}

/**
 * Sortable header component with 3-state sorting
 * States: null → asc → desc → null
 */
export function SortableHeader<TData>({ header, children }: SortableHeaderProps<TData>) {
  const { column } = header;
  const canSort = column.getCanSort();
  const isSorted = column.getIsSorted();

  if (!canSort) {
    return (
      <div className="flex items-center gap-1 truncate text-accent-12">
        {children || flexRender(column.columnDef.header, header.getContext())}
      </div>
    );
  }

  return (
    <button
      type="button"
      className={cn(
        "flex items-center gap-1 truncate text-accent-12",
        "cursor-pointer hover:text-accent-11 transition-colors",
        "focus:outline-none focus:text-accent-11",
      )}
      onClick={column.getToggleSortingHandler()}
      title={
        isSorted === "asc"
          ? "Sort descending"
          : isSorted === "desc"
            ? "Clear sorting"
            : "Sort ascending"
      }
    >
      <span>{children || flexRender(column.columnDef.header, header.getContext())}</span>
      <span className="flex-shrink-0">
        <SortIcon sorted={isSorted} />
      </span>
    </button>
  );
}

function SortIcon({ sorted }: { sorted: false | "asc" | "desc" }) {
  if (sorted === "asc") {
    return <CaretUp className="color-gray-9" iconSize="sm-thin" />;
  }

  if (sorted === "desc") {
    return <CaretDown className="color-gray-9" iconSize="sm-thin" />;
  }

  return <CaretExpandY className="color-gray-9" />;
}
