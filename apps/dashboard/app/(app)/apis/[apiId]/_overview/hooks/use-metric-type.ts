"use client";

import { parseAsStringLiteral, useQueryState } from "nuqs";

const METRIC_TYPES = ["requests", "creditSpend"] as const;
export type MetricType = (typeof METRIC_TYPES)[number];

export const METRIC_TYPE_LABELS: Record<MetricType, string> = {
  requests: "Requests",
  creditSpend: "Credit Spend",
};

export const useMetricType = () => {
  const [metricType, setMetricType] = useQueryState(
    "metricType",
    parseAsStringLiteral(METRIC_TYPES).withDefault("requests"),
  );

  return {
    metricType,
    setMetricType,
    isRequestsMode: metricType === "requests",
    isCreditSpendMode: metricType === "creditSpend",
  };
};
