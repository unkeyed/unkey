import { throttle } from "@/lib/utils";
import { useVirtualizer } from "@tanstack/react-virtual";
import { useCallback, useEffect } from "react";
import { TableConfig } from "../types";

export const useVirtualData = <T>({
  data,
  isLoading,
  config,
  onLoadMore,
  isFetchingNextPage,
  parentRef,
}: {
  data: T[];
  isLoading: boolean;
  config: TableConfig;
  onLoadMore?: () => void;
  isFetchingNextPage?: boolean;
  parentRef: React.RefObject<HTMLDivElement>;
}) => {
  const throttledLoadMore = useCallback(
    throttle(onLoadMore ?? (() => {}), config.throttleDelay, {
      leading: true,
      trailing: false,
    }),
    [onLoadMore, config.throttleDelay]
  );

  useEffect(() => {
    return () => {
      throttledLoadMore.cancel();
    };
  }, [throttledLoadMore]);

  return useVirtualizer({
    count: isLoading ? config.loadingRows : data.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => config.rowHeight,
    overscan: config.overscan,
    onChange: (instance) => {
      const lastItem = instance.getVirtualItems().at(-1);
      if (!lastItem || !onLoadMore) return;

      const scrollElement = instance.scrollElement;
      if (!scrollElement) return;

      const scrollOffset = scrollElement.scrollTop + scrollElement.clientHeight;
      const scrollThreshold = scrollElement.scrollHeight - config.rowHeight * 3;

      if (
        !isLoading &&
        !isFetchingNextPage &&
        lastItem.index >= data.length - 1 - instance.options.overscan &&
        scrollOffset >= scrollThreshold
      ) {
        throttledLoadMore();
      }
    },
  });
};
