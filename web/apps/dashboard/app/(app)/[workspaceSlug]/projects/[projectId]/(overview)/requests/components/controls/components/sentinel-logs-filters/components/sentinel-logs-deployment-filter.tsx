"use client";

import { FilterOperatorInput } from "@/components/logs/filter-operator-input";
import { useSentinelLogsFilters } from "../../../../../hooks/use-sentinel-logs-filters";

const OPTIONS = [{ id: "contains" as const, label: "contains" }];

export const SentinelDeploymentFilter = () => {
  const { filters, updateFilters } = useSentinelLogsFilters();

  const deploymentFilter = filters.find((f) => f.field === "deploymentId");
  const defaultText = deploymentFilter ? String(deploymentFilter.value) : "";

  const handleApply = (_operator: string, text: string) => {
    const otherFilters = filters.filter((f) => f.field !== "deploymentId");
    const newFilters = text
      ? [
          ...otherFilters,
          {
            id: crypto.randomUUID(),
            field: "deploymentId" as const,
            operator: "contains" as const,
            value: text,
          },
        ]
      : otherFilters;
    updateFilters(newFilters);
  };

  return (
    <FilterOperatorInput
      label="Deployment ID"
      options={OPTIONS}
      defaultOption="contains"
      defaultText={defaultText}
      onApply={handleApply}
    />
  );
};
