import { throttle } from "@/lib/utils";
import { type Virtualizer, useVirtualizer } from "@tanstack/react-virtual";
import { useCallback, useEffect, useMemo } from "react";
import type { DataTableConfig } from "../types";

interface UseVirtualizationProps {
  totalDataLength: number;
  isLoading: boolean;
  config: DataTableConfig;
  onLoadMore?: () => void;
  isFetchingNextPage?: boolean;
  parentRef: React.RefObject<HTMLDivElement | null>;
}

/**
 * TanStack Virtual integration with load more functionality
 */
export const useVirtualization = ({
  totalDataLength,
  isLoading,
  config,
  onLoadMore,
  isFetchingNextPage,
  parentRef,
}: UseVirtualizationProps) => {
  // Throttled load more callback
  const throttledFn = useMemo(
    () =>
      throttle(
        (...args: unknown[]) => {
          const cb = args[0] as (() => void) | undefined;
          cb?.();
        },
        config.throttleDelay,
        {
          leading: true,
          trailing: false,
        },
      ),
    [config.throttleDelay],
  );

  const throttledLoadMore = useCallback(() => {
    throttledFn(onLoadMore);
  }, [throttledFn, onLoadMore]);

  // Cleanup throttle on unmount
  useEffect(() => {
    return () => {
      throttledFn.cancel();
    };
  }, [throttledFn]);

  // Handle scroll and trigger load more
  const handleChange = useCallback(
    (instance: Virtualizer<HTMLDivElement, Element>) => {
      const lastItem = instance.getVirtualItems().at(-1);
      if (!lastItem || !onLoadMore) {
        return;
      }

      const scrollElement = instance.scrollElement;
      if (!scrollElement) {
        return;
      }

      // Calculate scroll position
      const scrollOffset = scrollElement.scrollTop + scrollElement.clientHeight;
      const scrollThreshold = scrollElement.scrollHeight - config.rowHeight * 3;

      // Trigger load more when near bottom
      if (
        !isLoading &&
        !isFetchingNextPage &&
        lastItem.index >= totalDataLength - 1 - instance.options.overscan &&
        scrollOffset >= scrollThreshold
      ) {
        throttledLoadMore();
      }
    },
    [
      isLoading,
      isFetchingNextPage,
      totalDataLength,
      config.rowHeight,
      throttledLoadMore,
      onLoadMore,
    ],
  );

  // Create virtualizer
  return useVirtualizer({
    count: isLoading ? config.loadingRows : totalDataLength,
    getScrollElement: useCallback(() => parentRef.current, [parentRef]),
    estimateSize: useCallback(() => config.rowHeight, [config.rowHeight]),
    overscan: config.overscan,
    onChange: handleChange,
  });
};
