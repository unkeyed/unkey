import { type Table, flexRender } from "@tanstack/react-table";
import { A11Y_LABELS, TABLE_CLASS_NAMES } from "../constants";

interface TableHeaderProps<TData> {
  table: Table<TData>;
  stickyHeader?: boolean;
  enableColumnResizing?: boolean;
}

export function TableHeader<TData>({
  table,
  stickyHeader = false,
  enableColumnResizing = false,
}: TableHeaderProps<TData>) {
  return (
    <thead
      className={`${TABLE_CLASS_NAMES.header} ${
        stickyHeader
          ? "sticky top-0 z-10 bg-gray-50 dark:bg-gray-900"
          : "bg-gray-50 dark:bg-gray-900"
      }`}
    >
      {table.getHeaderGroups().map((headerGroup) => (
        <tr key={headerGroup.id} className={TABLE_CLASS_NAMES.headerRow}>
          {headerGroup.headers.map((header) => {
            const canSort = header.column.getCanSort();
            const isSorted = header.column.getIsSorted();

            return (
              <th
                key={header.id}
                className={`
                  ${TABLE_CLASS_NAMES.headerCell}
                  relative
                  px-4 py-3
                  text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider
                  border-b border-gray-200 dark:border-gray-800
                  ${canSort ? `${TABLE_CLASS_NAMES.sortable} cursor-pointer select-none` : ""}
                  ${isSorted ? TABLE_CLASS_NAMES.sorted : ""}
                `}
                style={{
                  width: header.getSize(),
                  minWidth: header.column.columnDef.minSize,
                  maxWidth: header.column.columnDef.maxSize,
                }}
                onClick={canSort ? header.column.getToggleSortingHandler() : undefined}
                onKeyDown={
                  canSort
                    ? (e) => {
                        if (e.key === "Enter" || e.key === " ") {
                          e.preventDefault();
                          header.column.getToggleSortingHandler()?.(e);
                        }
                      }
                    : undefined
                }
                tabIndex={canSort ? 0 : undefined}
                role={canSort ? "button" : undefined}
                aria-sort={
                  isSorted === "asc" ? "ascending" : isSorted === "desc" ? "descending" : "none"
                }
              >
                <div className="flex items-center gap-2">
                  {header.isPlaceholder
                    ? null
                    : flexRender(header.column.columnDef.header, header.getContext())}

                  {/* Sort indicator */}
                  {canSort && (
                    <span className="flex flex-col">
                      {isSorted === "asc" && (
                        <svg
                          className="w-3 h-3 text-gray-900 dark:text-gray-100"
                          fill="currentColor"
                          viewBox="0 0 20 20"
                          aria-label={A11Y_LABELS.sortAscending}
                        >
                          <title>{A11Y_LABELS.sortAscending}</title>
                          <path d="M5 10l5-5 5 5H5z" />
                        </svg>
                      )}
                      {isSorted === "desc" && (
                        <svg
                          className="w-3 h-3 text-gray-900 dark:text-gray-100"
                          fill="currentColor"
                          viewBox="0 0 20 20"
                          aria-label={A11Y_LABELS.sortDescending}
                        >
                          <title>{A11Y_LABELS.sortDescending}</title>
                          <path d="M15 10l-5 5-5-5h10z" />
                        </svg>
                      )}
                      {!isSorted && (
                        <svg
                          className="w-3 h-3 text-gray-400 dark:text-gray-600"
                          fill="currentColor"
                          viewBox="0 0 20 20"
                          aria-label={A11Y_LABELS.sortNone}
                        >
                          <title>{A11Y_LABELS.sortNone}</title>
                          <path d="M5 8l5-5 5 5H5zM5 12l5 5 5-5H5z" />
                        </svg>
                      )}
                    </span>
                  )}
                </div>

                {/* Column resizer */}
                {enableColumnResizing && header.column.getCanResize() && (
                  <div
                    onMouseDown={header.getResizeHandler()}
                    onTouchStart={header.getResizeHandler()}
                    className={`
                      ${TABLE_CLASS_NAMES.resizer}
                      absolute
                      right-0
                      top-0
                      h-full
                      w-1
                      bg-gray-300
                      dark:bg-gray-700
                      cursor-col-resize
                      select-none
                      touch-none
                      opacity-0
                      hover:opacity-100
                      ${header.column.getIsResizing() ? "opacity-100 bg-primary-500" : ""}
                    `}
                  />
                )}
              </th>
            );
          })}
        </tr>
      ))}
    </thead>
  );
}
