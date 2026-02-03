"use client";

import {
  parseAsFilterValueArray,
  parseAsRelativeTime,
} from "@/components/logs/validation/utils/nuqs-parsers";
import {
  type RuntimeLogsFilterField,
  type RuntimeLogsFilterOperator,
  type RuntimeLogsFilterUrlValue,
  type RuntimeLogsFilterValue,
  type RuntimeLogsQuerySearchParams,
  runtimeLogsFilterFieldConfig,
} from "@/lib/schemas/runtime-logs.filter.schema";
import { parseAsInteger, useQueryStates } from "nuqs";
import { useCallback, useEffect, useMemo } from "react";

const parseAsFilterValArray = parseAsFilterValueArray<RuntimeLogsFilterOperator>([
  "is",
  "contains",
]);

export const queryParamsPayload = {
  severity: parseAsFilterValArray,
  message: parseAsFilterValArray,
  startTime: parseAsInteger,
  endTime: parseAsInteger,
  since: parseAsRelativeTime,
} as const;

export function useRuntimeLogsFilters() {
  const [searchParams, setSearchParams] = useQueryStates(queryParamsPayload, {
    history: "push",
  });

  // Initialize default "6h" filter on mount if no time filter exists
  useEffect(() => {
    if (
      searchParams.since === null &&
      searchParams.startTime === null &&
      searchParams.endTime === null
    ) {
      setSearchParams({ since: "6h" });
    }
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const filters = useMemo(() => {
    const activeFilters: RuntimeLogsFilterValue[] = [];

    searchParams.severity?.forEach((severity) => {
      activeFilters.push({
        id: crypto.randomUUID(),
        field: "severity",
        operator: severity.operator,
        value: severity.value,
        metadata: {
          colorClass: runtimeLogsFilterFieldConfig.severity.getColorClass?.(
            severity.value as string,
          ),
        },
      });
    });

    searchParams.message?.forEach((msg) => {
      activeFilters.push({
        id: crypto.randomUUID(),
        field: "message",
        operator: msg.operator,
        value: msg.value,
      });
    });

    ["startTime", "endTime", "since"].forEach((field) => {
      const value = searchParams[field as keyof RuntimeLogsQuerySearchParams];
      if (value !== null && value !== undefined) {
        activeFilters.push({
          id: crypto.randomUUID(),
          field: field as RuntimeLogsFilterField,
          operator: "is",
          value: value as string | number,
        });
      }
    });

    return activeFilters;
  }, [searchParams]);

  const updateFilters = useCallback(
    (newFilters: RuntimeLogsFilterValue[]) => {
      const newParams: Partial<RuntimeLogsQuerySearchParams> = {
        severity: null,
        message: null,
        startTime: null,
        endTime: null,
        since: null,
      };

      // Group filters by field
      const severityFilters: RuntimeLogsFilterUrlValue[] = [];
      const messageFilters: RuntimeLogsFilterUrlValue[] = [];

      newFilters.forEach((filter) => {
        switch (filter.field) {
          case "severity":
            severityFilters.push({
              value: filter.value,
              operator: filter.operator,
            });
            break;
          case "message":
            messageFilters.push({
              value: filter.value,
              operator: filter.operator,
            });
            break;
          case "startTime":
          case "endTime":
            newParams[filter.field] = filter.value as number;
            break;
          case "since":
            newParams.since = filter.value as string;
            break;
        }
      });

      // Set arrays to null when empty, otherwise use the filtered values
      newParams.severity = severityFilters.length > 0 ? severityFilters : null;
      newParams.message = messageFilters.length > 0 ? messageFilters : null;

      setSearchParams(newParams);
    },
    [setSearchParams],
  );

  const removeFilter = useCallback(
    (id: string) => {
      const newFilters = filters.filter((f) => f.id !== id);
      updateFilters(newFilters);
    },
    [filters, updateFilters],
  );

  return {
    filters,
    removeFilter,
    updateFilters,
  };
}
