"use client";

import type { RangePreset } from "@/lib/metrics/app-metrics";
import { createContext, use } from "react";

export const ChartRangeProvider = createContext<RangePreset | null>(null);

export function useIsRangePending(drawn: RangePreset): boolean {
  const wanted = use(ChartRangeProvider);
  return wanted !== null && wanted !== drawn;
}
