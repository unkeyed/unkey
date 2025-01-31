import { cn } from "@/lib/utils";
import type { VirtualItem } from "@tanstack/react-virtual";
import type { Column } from "../types";

const calculateGridTemplateColumns = (columns: Column<any>[]) => {
  return columns
    .map((column) => {
      if (typeof column.width === "number") {
        return `${column.width}px`;
      }

      if (typeof column.width === "string") {
        // Handle existing pixel and percentage values
        if (column.width.endsWith("px") || column.width.endsWith("%")) {
          return column.width;
        }
        if (column.width === "auto") {
          return "minmax(min-content, auto)";
        }
        if (column.width === "min") {
          return "min-content";
        }
        if (column.width === "1fr") {
          return "1fr";
        }
      }

      if (typeof column.width === "object") {
        if ("min" in column.width && "max" in column.width) {
          return `minmax(${column.width.min}px, ${column.width.max}px)`;
        }
        if ("flex" in column.width) {
          return `${column.width.flex}fr`;
        }
      }

      return "1fr"; // Default fallback
    })
    .join(" ");
};

export const TableRow = <T,>({
  item,
  columns,
  virtualRow,
  rowHeight,
  isSelected,
  rowClassName,
  selectedClassName,
  onClick,
  onRowClick,
  measureRef,
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
}) => {
  const gridTemplateColumns = calculateGridTemplateColumns(columns);

  return (
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
            nextElement.click();
          }
        }
        if (event.key === "ArrowUp" || event.key === "k") {
          event.preventDefault();
          const prevElement = document.querySelector(
            `[data-index="${virtualRow.index - 1}"]`,
          ) as HTMLElement;
          if (prevElement) {
            prevElement.focus();
            prevElement.click();
          }
        }
      }}
      ref={measureRef}
      onClick={onClick}
      className={cn(
        "grid text-xs cursor-pointer absolute top-0 left-0 w-full mt-1",
        "transition-all duration-75 ease-in-out",
        "group",
        rowClassName?.(item),
        selectedClassName?.(item, isSelected),
      )}
      style={{
        gridTemplateColumns,
        height: `${rowHeight}px`,
        position: "absolute",
        top: 0,
        left: 0,
        transform: `translateY(${virtualRow.start}px)`,
      }}
    >
      {columns.map((column) => (
        <div
          key={column.key}
          className={cn(
            "flex items-center h-full",
            "min-w-0", // Essential for truncation
            !column.noTruncate && "overflow-hidden", // Allow disabling truncation
          )}
          style={{
            minWidth: column.minWidth,
            maxWidth: column.maxWidth,
          }}
        >
          {column.noTruncate ? (
            column.render(item)
          ) : (
            <div className="w-full overflow-hidden">{column.render(item)}</div>
          )}
        </div>
      ))}
    </div>
  );
};
