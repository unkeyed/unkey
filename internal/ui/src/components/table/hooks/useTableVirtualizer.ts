"use client";
import type { Table } from "@tanstack/react-table";
import { useVirtualizer } from "@tanstack/react-virtual";
import { useRef } from "react";
import { DEFAULT_OVERSCAN, DEFAULT_ROW_HEIGHT } from "../constants";

interface UseTableVirtualizerProps<TData = unknown> {
  table: Table<TData>;
  enabled?: boolean;
  estimateSize?: number;
  overscan?: number;
  onLoadMore?: () => void;
  hasMore?: boolean;
}

export function useTableVirtualizer<TData = unknown>(props: UseTableVirtualizerProps<TData>) {
  const {
    table,
    enabled = true,
    estimateSize = DEFAULT_ROW_HEIGHT,
    overscan = DEFAULT_OVERSCAN,
    onLoadMore,
    hasMore = false,
  } = props;

  const parentRef = useRef<HTMLDivElement>(null);

  const virtualizer = useVirtualizer({
    count: enabled ? table.getRowModel().rows.length : 0,
    getScrollElement: () => parentRef.current,
    estimateSize: () => estimateSize,
    overscan,
    // Infinite scroll detection
    ...(onLoadMore && hasMore
      ? {
          onChange: (instance) => {
            const items = instance.getVirtualItems();
            if (items.length === 0) {
              return;
            }

            const lastItem = items[items.length - 1];
            if (!lastItem) {
              return;
            }

            // Trigger load more when we're near the end
            const threshold = Math.floor(instance.options.count * 0.8);
            if (lastItem.index >= threshold) {
              onLoadMore();
            }
          },
        }
      : {}),
  });

  const virtualRows = enabled ? virtualizer.getVirtualItems() : [];
  const totalSize = enabled ? virtualizer.getTotalSize() : 0;

  // Calculate padding for virtual scrolling
  const paddingTop = virtualRows.length > 0 ? (virtualRows[0]?.start ?? 0) : 0;
  const paddingBottom =
    virtualRows.length > 0 ? totalSize - (virtualRows[virtualRows.length - 1]?.end ?? 0) : 0;

  return {
    parentRef,
    virtualizer: enabled ? virtualizer : undefined,
    virtualRows,
    totalSize,
    paddingTop,
    paddingBottom,
  };
}
