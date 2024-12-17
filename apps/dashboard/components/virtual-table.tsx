import { Card, CardContent } from "@/components/ui/card";
import { cn, throttle } from "@/lib/utils";
import { useVirtualizer } from "@tanstack/react-virtual";
import { ScrollText } from "lucide-react";
import type React from "react";
import { useCallback, useEffect, useRef, useState } from "react";

export type Column<T> = {
  key: string;
  header: string;
  width: string;
  render: (item: T) => React.ReactNode;
};

export type VirtualTableProps<T> = {
  data: T[];
  columns: Column<T>[];
  isLoading?: boolean;
  rowHeight?: number;
  tableHeight?: string;
  onRowClick?: (item: T | null) => void;
  onLoadMore?: () => void;
  emptyState?: React.ReactNode;
  loadingRows?: number;
  overscanCount?: number;
  keyExtractor: (item: T) => string | number;
  rowClassName?: (item: T) => string;
  selectedClassName?: (item: T, isSelected: boolean) => string;
  selectedItem?: T | null;
  isFetchingNextPage?: boolean;
  renderDetails?: (
    item: T,
    onClose: () => void,
    distanceToTop: number
  ) => React.ReactNode;
};

const DEFAULT_ROW_HEIGHT = 26;
const DEFAULT_LOADING_ROWS = 50;
const DEFAULT_OVERSCAN = 5;
const TABLE_BORDER_THICKNESS = 1;
const THROTTLE_DELAY = 350;

export function VirtualTable<T>({
  data,
  columns,
  isLoading = false,
  rowHeight = DEFAULT_ROW_HEIGHT,
  tableHeight = "80vh",
  onRowClick,
  onLoadMore,
  emptyState,
  loadingRows = DEFAULT_LOADING_ROWS,
  overscanCount = DEFAULT_OVERSCAN,
  keyExtractor,
  rowClassName,
  selectedClassName,
  selectedItem,
  renderDetails,
  isFetchingNextPage,
}: VirtualTableProps<T>) {
  const parentRef = useRef<HTMLDivElement>(null);
  const tableRef = useRef<HTMLDivElement>(null);
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);

  const throttledLoadMore = useCallback(
    throttle(
      () => {
        if (onLoadMore) {
          onLoadMore();
        }
      },
      THROTTLE_DELAY,
      { leading: true, trailing: false }
    ),
    [onLoadMore]
  );

  // Cleanup throttle on unmount
  useEffect(() => {
    return () => {
      throttledLoadMore.cancel();
    };
  }, [throttledLoadMore]);

  const virtualizer = useVirtualizer({
    count: isLoading ? loadingRows : data.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => rowHeight,
    overscan: overscanCount,
    onChange: (instance) => {
      const lastItem = instance.getVirtualItems().at(-1);
      if (!lastItem || !onLoadMore) {
        return;
      }

      // Check if we're near the end and not currently loading
      if (
        !isLoading &&
        lastItem.index >= data.length - 1 - instance.options.overscan
      ) {
        throttledLoadMore();
      }
    },
  });

  const handleRowClick = (item: T) => {
    if (onRowClick) {
      onRowClick(item);
      setTableDistanceToTop(
        (tableRef.current?.getBoundingClientRect().top ?? 0) +
          window.scrollY -
          TABLE_BORDER_THICKNESS
      );
    }
  };

  const LoadingRow = () => (
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

  if (!isLoading && data.length === 0) {
    return (
      <div className="w-full">
        <div
          className="grid text-sm font-medium text-accent-12"
          style={{
            gridTemplateColumns: columns.map((col) => col.width).join(" "),
          }}
        >
          {columns.map((column) => (
            <div key={column.key} className="p-2 flex items-center">
              {column.header}
            </div>
          ))}
        </div>
        <div className="w-full border-t border-border" />

        <div className="flex justify-center items-center h-[600px] w-full">
          {emptyState || (
            <Card className="w-[400px] bg-background-subtle">
              <CardContent className="flex justify-center gap-2">
                <ScrollText />
                <div className="text-sm text-accent-12">No data available</div>
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className="w-full">
      <div
        className="grid text-sm font-medium text-accent-12"
        style={{
          gridTemplateColumns: columns.map((col) => col.width).join(" "),
        }}
      >
        {columns.map((column) => (
          <div key={column.key} className="p-2 flex items-center">
            {column.header}
          </div>
        ))}
      </div>
      <div className="w-full border-t border-border" />

      <div
        ref={(el) => {
          if (el) {
            //@ts-expect-error safe to bypass
            parentRef.current = el;
            //@ts-expect-error safe to bypass
            tableRef.current = el;
          }
        }}
        className="overflow-auto"
        style={{ height: tableHeight }}
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
                  <LoadingRow />
                </div>
              );
            }

            const item = data[virtualRow.index];
            if (!item) {
              return null;
            }

            const isSelected = selectedItem
              ? keyExtractor(selectedItem) === keyExtractor(item)
              : false;

            return (
              <div
                key={virtualRow.key}
                data-index={virtualRow.index}
                ref={virtualizer.measureElement}
                onClick={() => handleRowClick(item)}
                tabIndex={virtualRow.index}
                aria-selected={isSelected}
                onKeyDown={(event) => {
                  if (event.key === "Escape" && onRowClick) {
                    onRowClick(null);
                  }
                  if (event.key === "Enter" || event.key === " ") {
                    event.preventDefault();
                    handleRowClick(item);
                  }
                  if (event.key === "ArrowDown") {
                    event.preventDefault();
                    const nextElement = document.querySelector(
                      `[data-index="${virtualRow.index + 1}"]`
                    ) as HTMLElement;
                    nextElement?.focus();
                  }
                  if (event.key === "ArrowUp") {
                    event.preventDefault();
                    const prevElement = document.querySelector(
                      `[data-index="${virtualRow.index - 1}"]`
                    ) as HTMLElement;
                    prevElement?.focus();
                  }
                }}
                className={cn(
                  "grid text-[13px] leading-[14px] mb-[1px] rounded-[5px] cursor-pointer absolute top-0 left-0 w-full hover:bg-accent-3 pl-1 group",
                  rowClassName?.(item),
                  selectedItem && {
                    "opacity-50": !isSelected,
                    "opacity-100": isSelected,
                  },
                  selectedClassName?.(item, isSelected)
                )}
                style={{
                  gridTemplateColumns: columns
                    .map((col) => col.width)
                    .join(" "),
                  height: `${rowHeight}px`,
                  top: `${virtualRow.start}px`,
                }}
              >
                {columns.map((column) => (
                  <div key={column.key} className="px-[2px] flex items-center">
                    {column.render(item)}
                  </div>
                ))}
              </div>
            );
          })}
        </div>

        <div
          className={cn(
            "sticky bottom-0 py-2 bg-background/80 backdrop-blur-sm border-t border-border",
            "transition-opacity duration-300",
            isFetchingNextPage ? "opacity-100" : "opacity-0"
          )}
        >
          <div className="flex items-center justify-center gap-2 text-sm text-accent-11">
            <div className="h-1.5 w-1.5 rounded-full bg-accent-11 animate-pulse" />
            Loading more data
          </div>
        </div>

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
