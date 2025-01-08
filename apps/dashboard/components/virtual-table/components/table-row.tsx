import type { VirtualItem } from "@tanstack/react-virtual";
import { cn } from "@unkey/ui/src/lib/utils";
import type { Column } from "../types";

export const TableRow = <T,>({
  item,
  columns,
  virtualRow,
  rowHeight,
  isSelected,
  rowClassName,
  selectedClassName,
  onClick,
  measureRef,
  onRowClick,
}: {
  item: T;
  columns: Column<T>[];
  virtualRow: VirtualItem;
  rowHeight: number;
  isSelected: boolean;
  rowClassName?: (item: T) => string;
  selectedClassName?: (item: T, isSelected: boolean) => string;
  onClick: () => void;
  onRowClick?: (item: T | null) => void;
  measureRef: (element: HTMLElement | null) => void;
}) => (
  <div
    tabIndex={virtualRow.index}
    data-index={virtualRow.index}
    aria-selected={isSelected}
    onKeyDown={(event) => {
      if (event.key === "Escape") {
        event.preventDefault();
        onRowClick?.(null);
        const activeElement = document.activeElement as HTMLElement;
        activeElement?.blur();
      }

      if (event.key === "ArrowDown" || event.key === "j") {
        event.preventDefault();
        const nextElement = document.querySelector(
          `[data-index="${virtualRow.index + 1}"]`,
        ) as HTMLElement;
        if (nextElement) {
          nextElement.focus();
          nextElement.click(); // This will trigger onClick which calls handleRowClick
        }
      }
      if (event.key === "ArrowUp" || event.key === "k") {
        event.preventDefault();
        const prevElement = document.querySelector(
          `[data-index="${virtualRow.index - 1}"]`,
        ) as HTMLElement;
        if (prevElement) {
          prevElement.focus();
          prevElement.click(); // This will trigger onClick which calls handleRowClick
        }
      }
    }}
    ref={measureRef}
    onClick={onClick}
    className={cn(
      "grid text-xs cursor-pointer absolute top-0 left-0 w-full",
      "transition-all duration-75 ease-in-out",
      "hover:bg-accent-3 ",
      "group rounded-md",
      rowClassName?.(item),
      selectedClassName?.(item, isSelected),
    )}
    style={{
      gridTemplateColumns: columns.map((col) => col.width).join(" "),
      height: `${rowHeight}px`,
      top: `${virtualRow.start}px`,
    }}
  >
    {columns.map((column) => (
      <div key={column.key} className="truncate flex items-center h-full">
        {column.render(item)}
      </div>
    ))}
  </div>
);
