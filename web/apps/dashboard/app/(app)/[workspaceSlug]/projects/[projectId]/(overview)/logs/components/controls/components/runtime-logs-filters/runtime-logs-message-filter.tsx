"use client";

import { FilterOperatorInput } from "@/components/logs/filter-operator-input";
import { useRuntimeLogsFilters } from "../../../../hooks/use-runtime-logs-filters";

const OPTIONS = [{ id: "contains" as const, label: "contains" }];

export const RuntimeLogsMessageFilter = () => {
  const { filters, updateFilters } = useRuntimeLogsFilters();

  const messageFilter = filters.find((f) => f.field === "message");
  const defaultText = messageFilter ? String(messageFilter.value) : "";

  const handleApply = (_operator: string, text: string) => {
    const otherFilters = filters.filter((f) => f.field !== "message");
    const newFilters = text
      ? [
          ...otherFilters,
          {
            id: crypto.randomUUID(),
            field: "message" as const,
            operator: "contains" as const,
            value: text,
          },
        ]
      : otherFilters;
    updateFilters(newFilters);
  };

  return (
    <FilterOperatorInput
      label="Message"
      options={OPTIONS}
      defaultOption="contains"
      defaultText={defaultText}
      onApply={handleApply}
    />
  );
};
