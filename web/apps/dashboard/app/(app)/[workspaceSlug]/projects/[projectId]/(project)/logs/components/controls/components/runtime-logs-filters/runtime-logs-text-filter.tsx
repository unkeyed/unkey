"use client";

import { useRuntimeLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/logs/hooks/use-runtime-logs-filters";
import { FilterOperatorInput } from "@/components/logs/filter-operator-input";

const OPTIONS = [{ id: "contains" as const, label: "contains" }];

type RuntimeLogsTextFilterProps = {
  field: "message" | "attributes";
  label: string;
};

const validateIndexedText = (_operator: "contains", value: string): string | null =>
  value.length < 3 ? "Enter at least 3 characters." : null;

export const RuntimeLogsTextFilter = ({ field, label }: RuntimeLogsTextFilterProps) => {
  const { filters, updateFilters } = useRuntimeLogsFilters();

  const activeFilter = filters.find((filter) => filter.field === field);
  const defaultText = activeFilter ? String(activeFilter.value) : "";

  const handleApply = (_operator: string, text: string) => {
    const otherFilters = filters.filter((filter) => filter.field !== field);
    const newFilters = text
      ? [
          ...otherFilters,
          {
            id: crypto.randomUUID(),
            field,
            operator: "contains" as const,
            value: text,
          },
        ]
      : otherFilters;
    updateFilters(newFilters);
  };

  return (
    <FilterOperatorInput
      label={label}
      options={OPTIONS}
      defaultOption="contains"
      defaultText={defaultText}
      validate={validateIndexedText}
      onApply={handleApply}
    />
  );
};
