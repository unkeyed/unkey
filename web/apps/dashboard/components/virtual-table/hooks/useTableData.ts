import type { StripeSlot } from "@unkey/ui";
import type { SeparatorItem } from "../types";

export const useTableData = <TTableData>(
  realtimeData: readonly TTableData[],
  historicData: readonly TTableData[],
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

    stripeSlotAt: (index: number): StripeSlot => {
      if (index < realtimeData.length) {
        return { live: true, position: index };
      }
      return {
        live: false,
        position: realtimeData.length === 0 ? index : index - realtimeData.length - 1,
      };
    },

    getTotalLength: () => {
      return realtimeData.length + (realtimeData.length > 0 ? 1 : 0) + historicData.length;
    },
  };
};
