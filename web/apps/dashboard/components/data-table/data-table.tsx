import { cn } from "@/lib/utils";
import { flexRender } from "@tanstack/react-table";
import { useIsMobile } from "@unkey/ui";
import { Fragment, forwardRef, useImperativeHandle, useMemo, useRef } from "react";
import { LoadMoreFooter } from "./components/footer/load-more-footer";
import { SkeletonRow } from "./components/rows/skeleton-row";
import { EmptyState } from "./components/utils/empty-state";
import { RealtimeSeparator } from "./components/utils/realtime-separator";
import { DEFAULT_CONFIG, MOBILE_TABLE_HEIGHT } from "./constants/constants";
import { useDataTable } from "./hooks/use-data-table";
import { useRealtimeData } from "./hooks/use-realtime-data";
import { useTableHeight } from "./hooks/use-table-height";
import { useVirtualization } from "./hooks/use-virtualization";
import type { DataTableProps, SeparatorItem } from "./types";
import { calculateColumnWidth } from "./utils/column-width";

export type DataTableRef = {
  parentRef: HTMLDivElement | null;
  containerRef: HTMLDivElement | null;
};

/**
 * Main DataTable component with TanStack Table + TanStack Virtual
 */
function DataTableInner<TData>(
  props: DataTableProps<TData>,
  ref: React.Ref<DataTableRef>,
) {
  const {
    data: historicData,
    realtimeData = [],
    columns,
    getRowId,
    sorting,
    onSortingChange,
    rowSelection,
    onRowSelectionChange,
    onRowClick,
    onRowMouseEnter,
    onRowMouseLeave,
    selectedItem,
    onLoadMore,
    hasMore,
    isFetchingNextPage,
    config: userConfig,
    emptyState,
    loadMoreFooterProps,
    rowClassName,
    selectedClassName,
    fixedHeight: fixedHeightProp,
    enableKeyboardNav = true,
    enableSorting = true,
    enableRowSelection = false,
    isLoading = false,
    renderSkeletonRow,
  } = props;

  // Merge configs
  const config = { ...DEFAULT_CONFIG, ...userConfig };
  const isGridLayout = config.layout === "grid";

  // Refs
  const parentRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  // Mobile detection
  const isMobile = useIsMobile({ defaultValue: false });

  // Height calculation
  const calculatedHeight = useTableHeight(containerRef);
  const fixedHeight = fixedHeightProp ?? calculatedHeight;

  // Real-time data merging
  const tableDataHelper = useRealtimeData(getRowId, realtimeData, historicData);

  // TanStack Table
  const table = useDataTable({
    data: tableDataHelper.data,
    columns,
    getRowId,
    enableSorting,
    enableRowSelection,
    sorting,
    onSortingChange,
    rowSelection,
    onRowSelectionChange,
  });

  // TanStack Virtual
  const virtualizer = useVirtualization({
    totalDataLength: tableDataHelper.getTotalLength(),
    isLoading,
    config,
    onLoadMore,
    isFetchingNextPage,
    parentRef,
  });

  // Expose refs
  useImperativeHandle(
    ref,
    () => ({
      parentRef: parentRef.current,
      containerRef: containerRef.current,
    }),
    [],
  );

  // Calculate column widths
  const colWidths = useMemo(
    () => columns.map((col) => calculateColumnWidth(col.meta?.width)),
    [columns],
  );

  // CSS classes
  const hasPadding = config.containerPadding !== "px-0";

  const tableClassName = cn(
    "w-full",
    isGridLayout ? "border-collapse" : "border-separate border-spacing-0",
    config.tableLayout === "fixed" ? "table-fixed" : "table-auto",
  );

  const containerClassName = cn(
    "overflow-auto relative pb-4 bg-white dark:bg-black",
    config.containerPadding || "px-2",
  );

  // Empty state
  if (!isLoading && historicData.length === 0 && realtimeData.length === 0) {
    return (
      <div
        className="w-full flex flex-col md:h-full"
        style={{ height: `${fixedHeight}px` }}
        ref={containerRef}
      >
        <table className={tableClassName}>
          <colgroup>
            {columns.map((col, idx) => (
              <col key={col.id ?? idx} style={{ width: colWidths[idx] }} />
            ))}
          </colgroup>
          <thead className="sticky top-0 z-10 bg-white dark:bg-black">
            <tr>
              {table.getHeaderGroups()[0]?.headers.map((header) => (
                <th
                  key={header.id}
                  className={cn(
                    "text-sm font-medium text-accent-12 py-1 text-left",
                    header.column.columnDef.meta?.headerClassName,
                    header.column.columnDef.meta?.cellClassName,
                  )}
                >
                  {header.isPlaceholder ? null : (
                    <div className="truncate text-accent-12">
                      {flexRender(header.column.columnDef.header, header.getContext())}
                    </div>
                  )}
                </th>
              ))}
            </tr>
            <tr>
              <th colSpan={columns.length} className="p-0">
                <div className="w-full border-t border-gray-4" />
              </th>
            </tr>
          </thead>
        </table>
        {emptyState ? (
          <div className="flex-1 flex items-center justify-center">{emptyState}</div>
        ) : (
          <EmptyState />
        )}
      </div>
    );
  }

  // Keyboard navigation handler
  const handleKeyDown = (event: React.KeyboardEvent<HTMLTableRowElement>, rowIndex: number) => {
    if (!enableKeyboardNav) {
      return;
    }

    if (event.key === "Escape") {
      event.preventDefault();
      onRowClick?.(null);
      const activeElement = document.activeElement as HTMLElement;
      activeElement?.blur();
    }

    if (event.key === "ArrowDown" || event.key === "j") {
      event.preventDefault();
      const nextElement = document.querySelector(
        `[data-row-index="${rowIndex + 1}"]`,
      ) as HTMLElement;
      if (nextElement) {
        nextElement.focus();
        nextElement.click();
      }
    }

    if (event.key === "ArrowUp" || event.key === "k") {
      event.preventDefault();
      const prevElement = document.querySelector(
        `[data-row-index="${rowIndex - 1}"]`,
      ) as HTMLElement;
      if (prevElement) {
        prevElement.focus();
        prevElement.click();
      }
    }
  };

  // Main render
  return (
    <div className="w-full flex flex-col" ref={containerRef}>
      <div
        ref={parentRef}
        className={containerClassName}
        style={isMobile ? { height: MOBILE_TABLE_HEIGHT } : { height: `${fixedHeight}px` }}
      >
        <table className={tableClassName}>
          <colgroup>
            {columns.map((col, idx) => (
              <col key={col.id ?? idx} style={{ width: colWidths[idx] }} />
            ))}
          </colgroup>

          {/* Header */}
          <thead className="sticky top-0 z-10 bg-white dark:bg-black">
            {table.getHeaderGroups().map((headerGroup) => (
              <tr key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <th
                    key={header.id}
                    className={cn(
                      "text-sm font-medium text-accent-12 py-1 text-left relative",
                      header.column.columnDef.meta?.headerClassName,
                      header.column.columnDef.meta?.cellClassName,
                    )}
                  >
                    {header.isPlaceholder
                      ? null
                      : flexRender(header.column.columnDef.header, header.getContext())}
                  </th>
                ))}
              </tr>
            ))}
            <tr>
              <th colSpan={columns.length} className="p-0">
                <div className="relative w-full">
                  <div
                    className={cn(
                      "absolute border-t border-gray-4",
                      hasPadding ? "inset-x-[-8px]" : "inset-x-0",
                    )}
                  />
                </div>
              </th>
            </tr>
          </thead>

          {/* Body */}
          <tbody>
            {/* Top spacer */}
            <tr
              style={{
                height: `${virtualizer.getVirtualItems()[0]?.start || 0}px`,
              }}
            />

            {/* Virtual rows */}
            {virtualizer.getVirtualItems().map((virtualRow) => {
              // Loading skeleton
              if (isLoading) {
                return (
                  <tr
                    key={`skeleton-${virtualRow.key}`}
                    className={cn(config.rowBorders && "border-b border-gray-4")}
                    style={{ height: `${config.rowHeight}px` }}
                  >
                    {renderSkeletonRow ? (
                      renderSkeletonRow({
                        columns,
                        rowHeight: config.rowHeight,
                      })
                    ) : (
                      <SkeletonRow columns={columns} rowHeight={config.rowHeight} />
                    )}
                  </tr>
                );
              }

              // Get item
              const item = tableDataHelper.getItemAt(virtualRow.index);
              if (!item) {
                return null;
              }

              // Separator row
              const separator = item as SeparatorItem;
              if (separator.isSeparator) {
                return (
                  <Fragment key={`row-group-${virtualRow.key}`}>
                    <tr key={`spacer-${virtualRow.key}`} style={{ height: "4px" }} />
                    <tr key={`content-${virtualRow.key}`}>
                      <td colSpan={columns.length} className="p-0">
                        <RealtimeSeparator />
                      </td>
                    </tr>
                  </Fragment>
                );
              }

              // Data row
              const typedItem = item as TData;
              const rowId = getRowId(typedItem);
              const tableRow = table.getRowModel().rows.find((r) => r.id === rowId);

              if (!tableRow) {
                return null;
              }

              const isSelected = selectedItem ? getRowId(selectedItem) === rowId : false;

              // Grid layout (no spacing)
              if (isGridLayout) {
                return (
                  <tr
                    key={`content-${virtualRow.key}`}
                    tabIndex={virtualRow.index}
                    data-row-index={virtualRow.index}
                    aria-selected={isSelected}
                    onClick={() => onRowClick?.(typedItem)}
                    onMouseEnter={() => onRowMouseEnter?.(typedItem)}
                    onMouseLeave={() => onRowMouseLeave?.()}
                    onKeyDown={(e) => handleKeyDown(e, virtualRow.index)}
                    className={cn(
                      "cursor-pointer transition-colors hover:bg-accent/50 focus:outline-none focus:ring-1 focus:ring-opacity-40",
                      config.rowBorders && "border-b border-gray-4",
                      rowClassName?.(typedItem),
                      selectedClassName?.(typedItem, isSelected),
                    )}
                    style={{ height: `${config.rowHeight}px` }}
                  >
                    {tableRow.getVisibleCells().map((cell, idx) => (
                      <td
                        key={cell.id}
                        className={cn(
                          "text-xs align-middle whitespace-nowrap pr-4",
                          idx === 0 ? "rounded-l-md" : "",
                          idx === tableRow.getVisibleCells().length - 1 ? "rounded-r-md" : "",
                          cell.column.columnDef.meta?.cellClassName,
                        )}
                      >
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </td>
                    ))}
                  </tr>
                );
              }

              // Classic layout (with spacing)
              return (
                <Fragment key={`row-group-${virtualRow.key}`}>
                  {(config.rowSpacing ?? 4) > 0 && (
                    <tr
                      key={`spacer-${virtualRow.key}`}
                      style={{ height: `${config.rowSpacing ?? 4}px` }}
                    />
                  )}
                  <tr
                    key={`content-${virtualRow.key}`}
                    tabIndex={virtualRow.index}
                    data-row-index={virtualRow.index}
                    aria-selected={isSelected}
                    onClick={() => onRowClick?.(typedItem)}
                    onMouseEnter={() => onRowMouseEnter?.(typedItem)}
                    onMouseLeave={() => onRowMouseLeave?.()}
                    onKeyDown={(e) => handleKeyDown(e, virtualRow.index)}
                    className={cn(
                      "cursor-pointer transition-colors hover:bg-accent/50 focus:outline-none focus:ring-1 focus:ring-opacity-40",
                      config.rowBorders && "border-b border-gray-4",
                      rowClassName?.(typedItem),
                      selectedClassName?.(typedItem, isSelected),
                    )}
                    style={{ height: `${config.rowHeight}px` }}
                  >
                    {tableRow.getVisibleCells().map((cell, idx) => (
                      <td
                        key={cell.id}
                        className={cn(
                          "text-xs align-middle whitespace-nowrap pr-4",
                          idx === 0 ? "rounded-l-md" : "",
                          idx === tableRow.getVisibleCells().length - 1 ? "rounded-r-md" : "",
                          cell.column.columnDef.meta?.cellClassName,
                        )}
                      >
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </td>
                    ))}
                  </tr>
                </Fragment>
              );
            })}

            {/* Bottom spacer */}
            <tr
              style={{
                height: `${
                  virtualizer.getTotalSize() -
                  (virtualizer.getVirtualItems()[virtualizer.getVirtualItems().length - 1]?.end ||
                    0)
                }px`,
              }}
            />
          </tbody>
        </table>
      </div>
      {loadMoreFooterProps && (
        <LoadMoreFooter
          {...loadMoreFooterProps}
          onLoadMore={onLoadMore}
          isFetchingNextPage={isFetchingNextPage}
          hasMore={hasMore}
          totalVisible={virtualizer.getVirtualItems().length}
          totalCount={tableDataHelper.getTotalLength()}
        />
      )}
    </div>
  );
}

/**
 * Exported DataTable component with proper generic type support
 */
export const DataTable = forwardRef(DataTableInner) as <TData>(
  props: DataTableProps<TData> & { ref?: React.Ref<DataTableRef> },
) => React.ReactElement;
