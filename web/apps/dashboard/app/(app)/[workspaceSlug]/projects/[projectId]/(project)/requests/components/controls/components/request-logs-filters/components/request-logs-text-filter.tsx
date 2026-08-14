"use client";

import { useRequestLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/requests/hooks/use-request-logs-filters";
import { FilterOperatorInput } from "@/components/logs/filter-operator-input";
import type {
  RequestLogsFilterField,
  RequestLogsFilterOperator,
} from "@/lib/schemas/request-logs.filter.schema";

type RequestLogsTextFilterProps = {
  field: Extract<RequestLogsFilterField, "paths" | "host" | "requestId" | "region" | "userAgent">;
  label: string;
  operators: readonly RequestLogsFilterOperator[];
  validate?: (operator: RequestLogsFilterOperator, value: string) => string | null;
};

const operatorLabels: Record<RequestLogsFilterOperator, string> = {
  is: "is",
  startsWith: "starts with",
  contains: "contains",
};

export function RequestLogsTextFilter({
  field,
  label,
  operators,
  validate,
}: RequestLogsTextFilterProps) {
  const { filters, updateFilters } = useRequestLogsFilters();
  const activeFilter = filters.find((filter) => filter.field === field);
  const options = operators.map((operator) => ({ id: operator, label: operatorLabels[operator] }));

  return (
    <FilterOperatorInput
      label={label}
      options={options}
      defaultOption={activeFilter?.operator}
      defaultText={activeFilter ? String(activeFilter.value) : ""}
      validate={validate}
      onApply={(operator, value) => {
        const otherFilters = filters.filter((filter) => filter.field !== field);
        updateFilters([
          ...otherFilters,
          {
            id: crypto.randomUUID(),
            field,
            operator,
            value,
          },
        ]);
      }}
    />
  );
}
