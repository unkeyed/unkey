import { useMemo } from "react";
import type { SeparatorItem, TableDataItem } from "../types";

/**
 * Merges realtime and historic data with separator
 * Deduplicates by ID and inserts separator at boundary
 */
export const useRealtimeData = <TData>(
  getRowId: (row: TData) => string,
  realtimeData: TData[] = [],
  historicData: TData[] = [],
) => {
  return useMemo(() => {
    // If no realtime data, return historic data as-is
    if (realtimeData.length === 0) {
      return {
        data: historicData,
        getTotalLength: () => historicData.length,
        getItemAt: (index: number): TableDataItem<TData> => historicData[index],
      };
    }

    // Create ID set from realtime data for deduplication
    const realtimeIds = new Set(realtimeData.map(getRowId));

    // Filter out historic items that exist in realtime
    const filteredHistoric = historicData.filter((item) => !realtimeIds.has(getRowId(item)));

    // Total length: realtime + separator + deduplicated historic
    const totalLength = realtimeData.length + 1 + filteredHistoric.length;

    return {
      data: [...realtimeData, ...filteredHistoric],
      getTotalLength: () => totalLength,
      getItemAt: (index: number): TableDataItem<TData> => {
        // Realtime data
        if (index < realtimeData.length) {
          return realtimeData[index];
        }

        // Separator
        if (index === realtimeData.length) {
          return { isSeparator: true } as SeparatorItem;
        }

        // Historic data (offset by realtime length + separator)
        return filteredHistoric[index - realtimeData.length - 1];
      },
    };
  }, [realtimeData, historicData, getRowId]);
};
