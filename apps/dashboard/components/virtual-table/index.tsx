import { useCallback, useRef, useState } from "react";
import { DEFAULT_CONFIG } from "./constants";
import { useTableHeight } from "./hooks/useTableHeight";
import { useVirtualData } from "./hooks/useVirtualData";
import { VirtualTableProps } from "./types";
import { useScrollLock } from "usehooks-ts";
import { TableHeader } from "./components/table-header";
import { EmptyState } from "./components/empty-state";
import { cn } from "@unkey/ui/src/lib/utils";
import { LoadingRow } from "./components/loading-row";
import { TableRow } from "./components/table-row";
import { LoadingIndicator } from "./components/loading-indicator";

export function VirtualTable<T>({
  data,
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
  renderDetails,
  isFetchingNextPage,
}: VirtualTableProps<T>) {
  const config = { ...DEFAULT_CONFIG, ...userConfig };
  const parentRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);

  const fixedHeight = useTableHeight(
    containerRef,
    config.headerHeight,
    config.tableBorder
  );
  const virtualizer = useVirtualData({
    data,
    isLoading,
    config,
    onLoadMore,
    isFetchingNextPage,
    parentRef,
  });

  useScrollLock({
    autoLock: true,
    lockTarget:
      typeof window !== "undefined"
        ? (document.querySelector("#layout-wrapper") as HTMLElement)
        : undefined,
  });

  const handleRowClick = useCallback(
    (item: T) => {
      if (onRowClick) {
        onRowClick(item);
        setTableDistanceToTop(
          (parentRef.current?.getBoundingClientRect().top ?? 0) +
            window.scrollY -
            config.tableBorder
        );
      }
    },
    [onRowClick, config.tableBorder]
  );

  if (!isLoading && data.length === 0) {
    return (
      <div
        className="w-full h-full flex flex-col"
        ref={containerRef}
        style={{ height: `${fixedHeight}px` }}
      >
        <TableHeader columns={columns} />
        <EmptyState content={emptyState} />
      </div>
    );
  }

  return (
    <div className="w-full flex flex-col" ref={containerRef}>
      <TableHeader columns={columns} />
      <div
        ref={parentRef}
        data-table-container="true"
        className="overflow-auto pb-10 px-1 scroll-smooth"
        style={{ height: `${fixedHeight}px` }}
      >
        <div
          className={cn("transition-opacity duration-300", {
            "opacity-40": isLoading,
          })}
          style={{
            height: `${virtualizer.getTotalSize()}px`,
            width: "100%",
            position: "relative",
          }}
        >
          {virtualizer.getVirtualItems().map((virtualRow) => {
            if (isLoading) {
              return (
                <div
                  key={virtualRow.key}
                  style={{
                    position: "absolute",
                    top: `${virtualRow.start}px`,
                    width: "100%",
                  }}
                >
                  <LoadingRow columns={columns} />
                </div>
              );
            }

            const item = data[virtualRow.index];
            if (!item) return null;

            const isSelected = selectedItem
              ? keyExtractor(selectedItem) === keyExtractor(item)
              : false;

            return (
              <TableRow
                key={virtualRow.key}
                item={item}
                columns={columns}
                virtualRow={virtualRow}
                rowHeight={config.rowHeight}
                isSelected={isSelected}
                rowClassName={rowClassName}
                selectedClassName={selectedClassName}
                onClick={() => handleRowClick(item)}
                measureRef={virtualizer.measureElement}
                onRowClick={onRowClick}
              />
            );
          })}
        </div>

        {isFetchingNextPage && <LoadingIndicator />}

        {selectedItem &&
          renderDetails &&
          renderDetails(
            selectedItem,
            () => onRowClick?.(null as any),
            tableDistanceToTop
          )}
      </div>
    </div>
  );
}
