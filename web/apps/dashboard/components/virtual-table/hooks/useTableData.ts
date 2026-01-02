import type { SeparatorItem } from "../types";

export const useTableData = <TTableData>(
  realtimeData: TTableData[],
  historicData: TTableData[],
) => {
  return {
    getItemAt: (index: number): TTableData | SeparatorItem => {
      if (realtimeData.length === 0) {
        return historicData[index];
      }

      if (index < realtimeData.length) {
        return realtimeData[index];
      }

      if (index === realtimeData.length) {
        return { isSeparator: true };
      }

      return historicData[index - realtimeData.length - 1];
    },

    getTotalLength: () => {
      return realtimeData.length + (realtimeData.length > 0 ? 1 : 0) + historicData.length;
    },
  };
};
