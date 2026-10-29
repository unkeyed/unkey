import { usePageVirtualizer } from "@/hooks/use-page-virtualizer";
import { useCallback } from "react";

const ROW_HEIGHT = 48;

export function useVirtualList(rows: { id: string }[]) {
  const getItemKey = useCallback((index: number) => rows[index].id, [rows]);

  return usePageVirtualizer<Element>({
    count: rows.length,
    estimateSize: () => ROW_HEIGHT,
    overscan: 5,
    getItemKey,
  });
}
