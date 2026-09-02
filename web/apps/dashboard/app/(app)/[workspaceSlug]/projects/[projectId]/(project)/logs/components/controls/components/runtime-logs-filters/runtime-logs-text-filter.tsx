"use client";

import { useRuntimeLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/logs/hooks/use-runtime-logs-filters";
import { FilterOperatorInput } from "@/components/logs/filter-operator-input";
import {
  parseRuntimeLogsAttributeMatch,
  type RuntimeLogsFilterOperator,
} from "@/lib/schemas/runtime-logs.filter.schema";

const MESSAGE_OPTIONS = [{ id: "contains" as const, label: "contains" }];
const ATTRIBUTE_OPTIONS = [
  {
    id: "contains" as const,
    label: "contains",
    placeholder: "Enter attribute text",
  },
  {
    id: "is" as const,
    label: "matches",
    placeholder: "request.id = xyz",
  },
];

type RuntimeLogsTextFilterProps = {
  field: "message" | "attributes";
  label: string;
};

const validateIndexedText = (operator: RuntimeLogsFilterOperator, value: string): string | null => {
  if (operator === "is" && parseRuntimeLogsAttributeMatch(value) === null) {
    return "Use path = value with a value of at least 3 characters.";
  }
  return value.length < 3 ? "Enter at least 3 characters." : null;
};

export const RuntimeLogsTextFilter = ({ field, label }: RuntimeLogsTextFilterProps) => {
  const { filters, updateFilters } = useRuntimeLogsFilters();

  const activeFilter = filters.find((filter) => filter.field === field);
  const defaultText = activeFilter ? String(activeFilter.value) : "";
  const options = field === "attributes" ? ATTRIBUTE_OPTIONS : MESSAGE_OPTIONS;

  const handleApply = (operator: RuntimeLogsFilterOperator, text: string) => {
    const otherFilters = filters.filter((filter) => filter.field !== field);
    const newFilters = text
      ? [
          ...otherFilters,
          {
            id: crypto.randomUUID(),
            field,
            operator,
            value: text,
          },
        ]
      : otherFilters;
    updateFilters(newFilters);
  };

  return (
    <FilterOperatorInput<RuntimeLogsFilterOperator>
      label={label}
      options={options}
      defaultOption={activeFilter?.operator ?? "contains"}
      defaultText={defaultText}
      validate={validateIndexedText}
      onApply={handleApply}
    />
  );
};
