import { cn } from "@/lib/utils";
import { CaretDown, CaretExpandY, CaretUp, CircleCarretRight } from "@unkey/icons";
import { Fragment, type Ref, forwardRef, useImperativeHandle, useMemo, useRef } from "react";
import { EmptyState } from "./components/empty-state";
import { DEFAULT_CONFIG } from "./constants";
import { useTableData } from "./hooks/useTableData";
import { useTableHeight } from "./hooks/useTableHeight";
import { useVirtualData } from "./hooks/useVirtualData";
import type { Column, SeparatorItem, SortDirection, VirtualTableProps } from "./types";

const calculateTableLayout = (columns: Column<any>[]) => {
  return columns.map((column) => {
    let width = "auto";
    if (typeof column.width === "number") {
      width = `${column.width}px`;
    } else if (typeof column.width === "string") {
      width = column.width;
    } else if (typeof column.width === "object") {
      if ("min" in column.width && "max" in column.width) {
        width = `${column.width.min}px`;
      } else if ("flex" in column.width) {
        width = "auto";
      }
    }
    return { width };
  });
};

export type VirtualTableRef = {
  parentRef: HTMLDivElement | null;
  containerRef: HTMLDivElement | null;
};

export const VirtualTable = forwardRef<VirtualTableRef, VirtualTableProps<any>>(
  function VirtualTable<TTableData>(
    props: VirtualTableProps<TTableData>,
    ref: Ref<unknown> | undefined,
  ) {
    const {
      data: historicData,
      realtimeData = [],
      columns,
      isLoading = false,
      config: userConfig,
      onRowClick,
      onLoadMore,
      emptyState,
      keyExtractor,
      rowClassName,
      selectedClassName,
      selectedItem,
      isFetchingNextPage,
    } = props;

    // Merge configs, allowing specific overrides
    const config = { ...DEFAULT_CONFIG, ...userConfig };
    const isGridLayout = config.layoutMode === "grid";
    const parentRef = useRef<HTMLDivElement>(null);
    const containerRef = useRef<HTMLDivElement>(null);

    const fixedHeight = useTableHeight(containerRef, config.headerHeight, config.tableBorder);
    const tableData = useTableData<TTableData>(realtimeData, historicData);

    const virtualizer = useVirtualData({
      totalDataLength: tableData.getTotalLength(),
      isLoading,
      config,
      onLoadMore,
      isFetchingNextPage,
      parentRef,
    });

    const tableClassName = cn(
      "w-full",
      isGridLayout ? "border-collapse" : "border-separate border-spacing-0",
    );

    const containerClassName = cn(
      "overflow-auto relative",
      config.containerPadding || "px-2", // Default to px-2 if containerPadding is not specified
    );

    // Expose refs and methods to parent components. Primarily used for anchoring log details.
    useImperativeHandle(
      ref,
      () => ({
        parentRef: parentRef.current,
        containerRef: containerRef.current,
      }),
      [],
    );

    const colWidths = useMemo(() => calculateTableLayout(columns), [columns]);

    if (!isLoading && historicData.length === 0 && realtimeData.length === 0) {
      return (
        <div
          className="w-full flex flex-col h-full"
          style={{ height: `${fixedHeight}px` }}
          ref={containerRef}
        >
          <table className={tableClassName}>
            <thead className="sticky top-0 z-10 bg-background">
              <tr>
                {columns.map((column) => (
                  <th
                    key={column.key}
                    className={cn(
                      "text-sm font-medium text-accent-12 py-1 text-left",
                      column.headerClassName,
                    )}
                  >
                    <div className="truncate text-accent-12">{column.header}</div>
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
            <div className="flex-1 flex items-center justify-center h-full">{emptyState}</div>
          ) : (
            <EmptyState />
          )}
        </div>
      );
    }

    return (
      <div className="w-full flex flex-col" ref={containerRef}>
        <div ref={parentRef} className={containerClassName} style={{ height: `${fixedHeight}px` }}>
          <table className={tableClassName}>
            <colgroup>
              {colWidths.map((col, idx) => (
                // biome-ignore lint/suspicious/noArrayIndexKey: <explanation>
                <col key={idx} style={{ width: col.width }} />
              ))}
            </colgroup>

            <thead className="sticky top-0 z-10">
              <tr>
                <th colSpan={columns.length} className="p-0">
                  <div className="absolute inset-x-[-8px] top-0 bottom-[0px] bg-gray-1" />
                </th>
              </tr>
              <tr>
                {columns.map((column) => (
                  <th
                    key={column.key}
                    className={cn(
                      "text-sm font-medium text-accent-12 py-1 text-left relative",
                      column.headerClassName,
                    )}
                  >
                    <HeaderCell column={column} />
                  </th>
                ))}
              </tr>
              <tr>
                <th colSpan={columns.length} className="p-0">
                  <div className="relative w-full">
                    <div className="absolute inset-x-[-8px] border-t border-gray-4" />
                  </div>
                </th>
              </tr>
            </thead>

            <tbody>
              <tr
                style={{
                  height: `${virtualizer.getVirtualItems()[0]?.start || 0}px`,
                }}
              />
              {virtualizer.getVirtualItems().map((virtualRow) => {
                if (isLoading) {
                  return (
                    <tr key={virtualRow.key} style={{ height: `${config.rowHeight}px` }}>
                      {columns.map((column) => (
                        <td key={column.key} className="pr-4">
                          <div className="h-4 bg-accent-3 rounded animate-pulse" />
                        </td>
                      ))}
                    </tr>
                  );
                }
                const item = tableData.getItemAt(virtualRow.index);
                if (!item) {
                  return null;
                }

                const separator = item as SeparatorItem;
                if (separator.isSeparator) {
                  return (
                    <tr key={`content-${virtualRow.key}`}>
                      <td colSpan={columns.length} className="p-0">
                        <div className="h-[26px] bg-info-2 font-mono text-xs text-info-11 rounded-md flex items-center gap-3 px-2">
                          <CircleCarretRight className="size-3" />
                          Live
                        </div>
                      </td>
                    </tr>
                  );
                }

                const typedItem = item as TTableData;
                const isSelected = selectedItem
                  ? keyExtractor(selectedItem) === keyExtractor(typedItem)
                  : false;

                if (isGridLayout) {
                  // Grid layout: single row with optional border
                  return (
                    <tr
                      key={`content-${virtualRow.key}`}
                      tabIndex={virtualRow.index}
                      data-index={virtualRow.index}
                      aria-selected={isSelected}
                      onClick={() => onRowClick?.(typedItem)}
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
                      className={cn(
                        "cursor-pointer transition-colors hover:bg-accent/50 focus:outline-none focus:ring-1 focus:ring-opacity-40",
                        config.rowBorders && "border-b border-gray-4",
                        rowClassName?.(typedItem),
                        selectedClassName?.(typedItem, isSelected),
                      )}
                      style={{ height: `${config.rowHeight}px` }}
                    >
                      {columns.map((column, idx) => (
                        <td
                          key={column.key}
                          className={cn(
                            "text-xs align-middle whitespace-nowrap pr-4",
                            idx === 0 ? "rounded-l-md" : "",
                            idx === columns.length - 1 ? "rounded-r-md" : "",
                          )}
                        >
                          {column.render(typedItem)}
                        </td>
                      ))}
                    </tr>
                  );
                }
                // Classic layout: fragment with configurable spacer row
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
                      data-index={virtualRow.index}
                      aria-selected={isSelected}
                      onClick={() => onRowClick?.(typedItem)}
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
                      className={cn(
                        "cursor-pointer transition-colors hover:bg-accent/50 focus:outline-none focus:ring-1 focus:ring-opacity-40",
                        config.rowBorders && "border-b border-gray-4", // Still allow borders in classic mode
                        rowClassName?.(typedItem),
                        selectedClassName?.(typedItem, isSelected),
                      )}
                      style={{ height: `${config.rowHeight}px` }}
                    >
                      {columns.map((column, idx) => (
                        <td
                          key={column.key}
                          className={cn(
                            "text-xs align-middle whitespace-nowrap pr-4",
                            idx === 0 ? "rounded-l-md" : "",
                            idx === columns.length - 1 ? "rounded-r-md" : "",
                          )}
                        >
                          {column.render(typedItem)}
                        </td>
                      ))}
                    </tr>
                  </Fragment>
                );
              })}
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
          <LoadMoreFooter
            onLoadMore={onLoadMore}
            isFetchingNextPage={isFetchingNextPage}
            totalVisible={virtualizer.getVirtualItems().length}
            totalCount={tableData.getTotalLength()}
          />
        </div>
      </div>
    );
  },
);

function SortIcon({ direction }: { direction?: SortDirection | null }) {
  if (!direction) {
    return <CaretExpandY className="color-gray-9" />;
  }
  return direction === "asc" ? (
    <CaretUp className="color-gray-9" size="sm-thin" />
  ) : (
    <CaretDown className="color-gray-9" size="sm-thin" />
  );
}

function HeaderCell<T>({ column }: { column: Column<T> }) {
  const { direction, onSort, sortable } = column.sort ?? {};
  const handleSort = () => {
    if (!sortable || !onSort) {
      return;
    }

    const nextDirection = !direction ? "asc" : direction === "asc" ? "desc" : null;

    onSort(nextDirection);
  };

  return (
    // biome-ignore lint/a11y/useKeyWithClickEvents: <explanation>
    <div
      className={cn(
        "flex items-center gap-1 truncate text-accent-12",
        sortable && "cursor-pointer",
      )}
      onClick={sortable ? handleSort : undefined}
    >
      <span>{column.header}</span>
      {sortable && (
        <span className="flex-shrink-0">
          <SortIcon direction={direction} />
        </span>
      )}
    </div>
  );
}
