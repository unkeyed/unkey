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
import { useCallback, useMemo } from "react";

const parseAsFilterValArray = parseAsFilterValueArray<RuntimeLogsFilterOperator>([
  "is",
  "contains",
]);

const arrayFields = [
  "severity",
  "message",
  "attributes",
  "appId",
  "environmentId",
  "deploymentId",
  "region",
  "instanceId",
] as const;
const timeFields = ["startTime", "endTime", "since"] as const;

export const queryParamsPayload = {
  severity: parseAsFilterValArray,
  message: parseAsFilterValArray,
  attributes: parseAsFilterValArray,
  appId: parseAsFilterValArray,
  environmentId: parseAsFilterValArray,
  deploymentId: parseAsFilterValArray,
  region: parseAsFilterValArray,
  instanceId: parseAsFilterValArray,
  startTime: parseAsInteger,
  endTime: parseAsInteger,
  since: parseAsRelativeTime,
} as const;

export function useRuntimeLogsFilters() {
  const [searchParams, setSearchParams] = useQueryStates(queryParamsPayload, {
    history: "push",
  });

  const filters = useMemo(() => {
    const activeFilters: RuntimeLogsFilterValue[] = [];

    arrayFields.forEach((field) => {
      const getColorClass = runtimeLogsFilterFieldConfig[field].getColorClass;
      searchParams[field]?.forEach((item) => {
        activeFilters.push({
          id: crypto.randomUUID(),
          field,
          operator: item.operator,
          value: item.value,
          metadata: getColorClass ? { colorClass: getColorClass(item.value as string) } : undefined,
        });
      });
    });

    timeFields.forEach((field) => {
      const value = searchParams[field];
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
      const newParams: Partial<RuntimeLogsQuerySearchParams> = Object.fromEntries([
        ...arrayFields.map((field) => [field, null]),
        ...timeFields.map((field) => [field, null]),
      ]);

      const filterGroups = arrayFields.reduce(
        (acc, field) => {
          acc[field] = [];
          return acc;
        },
        {} as Record<(typeof arrayFields)[number], RuntimeLogsFilterUrlValue[]>,
      );

      newFilters.forEach((filter) => {
        if (arrayFields.includes(filter.field as (typeof arrayFields)[number])) {
          filterGroups[filter.field as (typeof arrayFields)[number]].push({
            value: filter.value,
            operator: filter.operator,
          });
        } else if (filter.field === "startTime" || filter.field === "endTime") {
          newParams[filter.field] = filter.value as number;
        } else if (filter.field === "since") {
          newParams.since = filter.value as string;
        }
      });

      arrayFields.forEach((field) => {
        newParams[field] = filterGroups[field].length > 0 ? filterGroups[field] : null;
      });

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
