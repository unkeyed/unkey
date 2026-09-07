"use client";

import { trpc } from "@/lib/trpc/client";
import type { Router } from "@/lib/trpc/routers";
import type { inferRouterOutputs } from "@trpc/server";
import { useMemo } from "react";

export type Delivery = inferRouterOutputs<Router>["logdrain"]["recentDeliveries"][number];
export const WINDOW_HOURS = 24;

export function isFailure(delivery: Delivery): boolean {
  return delivery.outcome !== "success";
}

export function detailText(delivery: Delivery): string {
  return (delivery.responseBody || delivery.error).replace(/\s+/g, " ").trim();
}

export function useDeliveries(drainId: string) {
  const query = trpc.logdrain.recentDeliveries.useQuery({ drainId });
  return { deliveries: query.data, isError: query.isError, retry: () => query.refetch() };
}

type DeliveryTotals = {
  delivered: number;
  failed: number;
  successRate: number | null;
  lastDeliveryMs: number | null;
};

export function useDeliveryTotals(drainId: string) {
  const { data, isError, refetch } = trpc.logdrain.metrics.useQuery({
    drainId,
    hours: WINDOW_HOURS,
  });

  const totals = useMemo((): DeliveryTotals | null => {
    if (!data) {
      return null;
    }
    let delivered = 0;
    let failed = 0;
    let lastDeliveryMs: number | null = null;
    for (const point of data.series) {
      delivered += point.successCount;
      failed += point.transientErrorCount + point.permanentErrorCount;
      if (point.lastSuccessMs > 0) {
        lastDeliveryMs = Math.max(lastDeliveryMs ?? 0, point.lastSuccessMs);
      }
    }
    const attempts = delivered + failed;
    return {
      delivered,
      failed,
      lastDeliveryMs,
      successRate: attempts === 0 ? null : (delivered / attempts) * 100,
    };
  }, [data]);

  return { totals, isError, retry: () => refetch() };
}
