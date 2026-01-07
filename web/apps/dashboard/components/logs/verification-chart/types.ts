import type { TimeseriesGranularity } from "@/lib/trpc/routers/utils/granularity";

export type TimeseriesData = {
  displayX: string;
  originalTimestamp: number;
  valid: number;
  total: number;
  success: number;
  error: number;
  [key: string]: string | number;
};

export type Filter = {
  field: string;
  value: string | number;
  id: string;
  operator: string;
};

export type ChartLabels = {
  title: string;
  primaryLabel: string;
  primaryKey: string;
  secondaryLabel: string;
  secondaryKey: string;
};

export type ChartTooltipItem = {
  label: string;
  dataKey: string;
};

export type DataHook = () => {
  timeseries: TimeseriesData[];
  isLoading: boolean;
  isError: boolean;
  granularity?: TimeseriesGranularity;
};

export type FiltersHook = () => {
  filters: Filter[];
  updateFilters: (filters: Filter[]) => void;
};