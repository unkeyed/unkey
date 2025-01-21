import { cn } from "@/lib/utils";
import { CircleCarretRight } from "@unkey/icons";
import { useCallback, useRef, useState } from "react";
import { useScrollLock } from "usehooks-ts";
import { EmptyState } from "./components/empty-state";
import { LoadingIndicator } from "./components/loading-indicator";
import { LoadingRow } from "./components/loading-row";
import { TableHeader } from "./components/table-header";
import { TableRow } from "./components/table-row";
import { DEFAULT_CONFIG } from "./constants";
import { useTableHeight } from "./hooks/useTableHeight";
import { useVirtualData } from "./hooks/useVirtualData";
import type { VirtualTableProps } from "./types";

const SEPARATOR_HEIGHT = 26;

export function VirtualTable<T>({
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
  renderDetails,
  isFetchingNextPage,
}: VirtualTableProps<T>) {
  const config = { ...DEFAULT_CONFIG, ...userConfig };
  const parentRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);

  const fixedHeight = useTableHeight(containerRef, config.headerHeight, config.tableBorder);
  const historicVirtualizer = useVirtualData({
    data: historicData,
    isLoading,
    config,
    onLoadMore,
    isFetchingNextPage,
    parentRef,
  });

  const realtimeVirtualizer = useVirtualData({
    data: realtimeData,
    isLoading: false,
    config,
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
            config.tableBorder,
        );
      }
    },
    [onRowClick, config.tableBorder],
  );

  if (!isLoading && historicData.length === 0 && realtimeData.length === 0) {
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

  // Calculate total height including both sections and separator if needed
  const totalHeight =
    realtimeVirtualizer.getTotalSize() +
    historicVirtualizer.getTotalSize() +
    (realtimeData.length > 0 ? SEPARATOR_HEIGHT : 0);

  return (
    <div className="w-full flex flex-col" ref={containerRef}>
      <TableHeader columns={columns} />
      <div
        ref={parentRef}
        data-table-container="true"
        className="overflow-auto pb-2 px-1 scroll-smooth relative"
        style={{ height: `${fixedHeight}px` }}
      >
        <div
          className={cn("transition-opacity duration-300", {
            "opacity-40": isLoading,
          })}
          style={{
            height: `${totalHeight}px`,
            width: "100%",
            position: "relative",
          }}
        >
          {/* Render real-time data section */}
          {realtimeVirtualizer.getVirtualItems().map((virtualRow) => {
            const item = realtimeData[virtualRow.index];
            if (!item) {
              return null;
            }

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
                selectedClassName={selectedClassName}
                rowClassName={rowClassName}
                onClick={() => handleRowClick(item)}
                measureRef={realtimeVirtualizer.measureElement}
                onRowClick={onRowClick}
              />
            );
          })}

          {/* Separator */}
          {realtimeData.length > 0 && (
            <div
              className="w-full sticky"
              style={{
                top: `${realtimeVirtualizer.getTotalSize()}px`,
                zIndex: 10,
              }}
            >
              <div className="h-[26px] bg-info-2 font-mono text-xs text-info-11 rounded-md flex items-center gap-3 px-2">
                <CircleCarretRight className="size-3" />
                Live
              </div>
            </div>
          )}

          {/* Render historic data section */}
          {historicVirtualizer.getVirtualItems().map((virtualRow) => {
            if (isLoading) {
              return (
                <div
                  key={virtualRow.key}
                  style={{
                    position: "absolute",
                    top: `${
                      virtualRow.start +
                      realtimeVirtualizer.getTotalSize() +
                      (realtimeData.length > 0 ? SEPARATOR_HEIGHT : 0)
                    }px`,
                    width: "100%",
                  }}
                >
                  <LoadingRow columns={columns} />
                </div>
              );
            }

            const item = historicData[virtualRow.index];
            if (!item) {
              return null;
            }

            const isSelected = selectedItem
              ? keyExtractor(selectedItem) === keyExtractor(item)
              : false;

            return (
              <TableRow
                key={virtualRow.key}
                item={item}
                columns={columns}
                virtualRow={{
                  ...virtualRow,
                  start:
                    virtualRow.start +
                    (realtimeData.length > 0
                      ? realtimeVirtualizer.getTotalSize() + SEPARATOR_HEIGHT
                      : 0),
                }}
                rowHeight={config.rowHeight}
                isSelected={isSelected}
                rowClassName={rowClassName}
                selectedClassName={selectedClassName}
                onClick={() => handleRowClick(item)}
                measureRef={historicVirtualizer.measureElement}
                onRowClick={onRowClick}
              />
            );
          })}
        </div>

        {isFetchingNextPage && <LoadingIndicator />}

        {selectedItem &&
          renderDetails &&
          renderDetails(selectedItem, () => onRowClick?.(null), tableDistanceToTop)}
      </div>
    </div>
  );
}
