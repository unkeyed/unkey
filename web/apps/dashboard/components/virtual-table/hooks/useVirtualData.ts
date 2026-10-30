import { usePageVirtualizer } from "@/hooks/use-page-virtualizer";
import { throttle } from "@/lib/utils";
import type { Virtualizer } from "@tanstack/react-virtual";
import { useCallback, useEffect, useMemo } from "react";
import type { TableConfig } from "../types";

export const useVirtualData = ({
  totalDataLength,
  isLoading,
  config,
  onLoadMore,
  isFetchingNextPage,
  getItemSize,
}: {
  totalDataLength: number;
  isLoading: boolean;
  config: TableConfig;
  onLoadMore?: () => void;
  isFetchingNextPage?: boolean;
  getItemSize?: (index: number) => number;
}) => {
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

  useEffect(() => {
    return () => {
      throttledFn.cancel();
    };
  }, [throttledFn]);

  const handleChange = useCallback(
    (instance: Virtualizer<Window, Element>) => {
      const lastItem = instance.getVirtualItems().at(-1);
      if (!lastItem || !onLoadMore || !instance.scrollRect) {
        return;
      }

      const scrollOffset = (instance.scrollOffset ?? 0) + instance.scrollRect.height;
      const scrollThreshold =
        instance.options.scrollMargin + instance.getTotalSize() - config.rowHeight * 3;

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

  return usePageVirtualizer<Element>({
    count: isLoading ? config.loadingRows : totalDataLength,
    estimateSize: useCallback(
      (index: number) => (getItemSize ? getItemSize(index) : config.rowHeight),
      [config.rowHeight, getItemSize],
    ),
    overscan: config.overscan,
    onChange: handleChange,
    gap: 4,
  });
};
