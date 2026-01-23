import { throttle } from "@/lib/utils";
import { type Virtualizer, useVirtualizer } from "@tanstack/react-virtual";
import { useCallback, useEffect, useMemo } from "react";
import type { TableConfig } from "../types";

export const useVirtualData = ({
  totalDataLength,
  isLoading,
  config,
  onLoadMore,
  isFetchingNextPage,
  parentRef,
}: {
  totalDataLength: number;
  isLoading: boolean;
  config: TableConfig;
  onLoadMore?: () => void;
  isFetchingNextPage?: boolean;
  parentRef: React.RefObject<HTMLDivElement | null>;
}) => {
  const throttledFn = useMemo(
    () =>
      throttle((cb?: () => void) => cb?.(), config.throttleDelay, {
        leading: true,
        trailing: false,
      }),
    [config.throttleDelay],
  );

  const throttledLoadMore = useCallback(() => {
    throttledFn(onLoadMore);
  }, [throttledFn, onLoadMore]);

  useEffect(() => {
    return () => {
      throttledFn.cancel();
    };
  }, [throttledFn]);

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

      const scrollOffset = scrollElement.scrollTop + scrollElement.clientHeight;
      const scrollThreshold = scrollElement.scrollHeight - config.rowHeight * 3;

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

  return useVirtualizer({
    count: isLoading ? config.loadingRows : totalDataLength,
    getScrollElement: useCallback(() => parentRef.current, [parentRef]),
    estimateSize: useCallback(() => config.rowHeight, [config.rowHeight]),
    overscan: config.overscan,
    onChange: handleChange,
    gap: 4, // Add this line
  });
};
