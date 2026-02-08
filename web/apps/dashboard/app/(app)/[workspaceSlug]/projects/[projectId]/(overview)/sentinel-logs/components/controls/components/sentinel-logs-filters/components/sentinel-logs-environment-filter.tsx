"use client";

import { FilterOperatorInput } from "@/components/logs/filter-operator-input";
import { useSentinelLogsFilters } from "../../../../../hooks/use-sentinel-logs-filters";

const OPTIONS = [{ id: "contains" as const, label: "contains" }];

export const SentinelEnvironmentFilter = () => {
  const { filters, updateFilters } = useSentinelLogsFilters();

  const environmentFilter = filters.find((f) => f.field === "environmentId");
  const defaultText = environmentFilter ? String(environmentFilter.value) : "";

  const handleApply = (_operator: string, text: string) => {
    const otherFilters = filters.filter((f) => f.field !== "environmentId");
    const newFilters = text
      ? [
          ...otherFilters,
          {
            id: crypto.randomUUID(),
            field: "environmentId" as const,
            operator: "contains" as const,
            value: text,
          },
        ]
      : otherFilters;
    updateFilters(newFilters);
  };

  return (
    <FilterOperatorInput
      label="Environment ID"
      options={OPTIONS}
      defaultOption="contains"
      defaultText={defaultText}
      onApply={handleApply}
    />
  );
};
