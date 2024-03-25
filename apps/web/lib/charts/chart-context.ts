"use client";
import { createContext, useContext } from "react";
import {
  type ChartContext as ChartContextType,
  type ChartTooltipContext as ChartTooltipContextType,
  Datum,
} from "./types";

export const ChartContext = createContext<ChartContextType | null>(null);

export function useChartContext<T extends Datum>(): ChartContextType<T> {
  const chartContext = useContext(ChartContext);
  if (!chartContext) {
    throw new Error("No chart context");
  }
  return chartContext;
}

export const ChartTooltipContext = createContext<ChartTooltipContextType | null>(null);

export function useChartTooltipContext<T extends Datum>(): ChartTooltipContextType<T> {
  const chartTooltipContext = useContext(ChartTooltipContext);
  if (!chartTooltipContext) {
    throw new Error("No chart tooltip context");
  }
  return chartTooltipContext;
}
