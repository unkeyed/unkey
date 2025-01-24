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
import { useTableData } from "./hooks/useTableData";
import { useTableHeight } from "./hooks/useTableHeight";
import { useVirtualData } from "./hooks/useVirtualData";
import type { VirtualTableProps } from "./types";

export function VirtualTable<TTableData>({
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
}: VirtualTableProps<TTableData>) {
  const config = { ...DEFAULT_CONFIG, ...userConfig };
  const parentRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);

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

  useScrollLock({
    autoLock: true,
    lockTarget:
      typeof window !== "undefined"
        ? (document.querySelector("#layout-wrapper") as HTMLElement)
        : undefined,
  });

  const handleRowClick = useCallback(
    (item: TTableData) => {
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

            const item = tableData.getItemAt(virtualRow.index);
            if (!item) {
              return null;
            }

            // Render separator
            //@ts-expect-error This is our hacky way to separate live data from historic data. This separator acts as just another item in the data list to preserve the correct start position.
            if ("isSeparator" in item && item.isSeparator) {
              return (
                <div
                  key={virtualRow.key}
                  className="w-full mt-1"
                  style={{
                    position: "absolute",
                    top: `${virtualRow.start + virtualRow.index * 4}px`, // Add 4px gap per row
                  }}
                >
                  <div className="h-[26px] bg-info-2 font-mono text-xs text-info-11 rounded-md flex items-center gap-3 px-2">
                    <CircleCarretRight className="size-3" />
                    Live
                  </div>
                </div>
              );
            }

            // Regular row
            const typedItem = item as TTableData;
            const isSelected = selectedItem
              ? keyExtractor(selectedItem) === keyExtractor(typedItem)
              : false;

            return (
              <TableRow
                key={virtualRow.key}
                item={typedItem}
                columns={columns}
                virtualRow={virtualRow}
                rowHeight={config.rowHeight}
                isSelected={isSelected}
                selectedClassName={selectedClassName}
                rowClassName={rowClassName}
                onClick={() => handleRowClick(typedItem)}
                measureRef={virtualizer.measureElement}
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
