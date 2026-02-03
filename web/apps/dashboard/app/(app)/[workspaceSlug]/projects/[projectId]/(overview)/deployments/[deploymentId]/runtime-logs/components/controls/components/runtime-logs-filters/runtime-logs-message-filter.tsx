"use client";

import { FilterOperatorInput } from "@/components/logs/filter-operator-input";
import { useRuntimeLogsFilters } from "../../../../hooks/use-runtime-logs-filters";

const OPTIONS = [{ id: "contains" as const, label: "contains" }];

export const RuntimeLogsMessageFilter = () => {
  const { queryParams, updateFilters } = useRuntimeLogsFilters();

  const handleApply = (_operator: string, text: string) => {
    updateFilters({ message: text });
  };

  return (
    <FilterOperatorInput
      label="Message"
      options={OPTIONS}
      defaultOption="contains"
      defaultText={queryParams.message}
      onApply={handleApply}
    />
  );
};
