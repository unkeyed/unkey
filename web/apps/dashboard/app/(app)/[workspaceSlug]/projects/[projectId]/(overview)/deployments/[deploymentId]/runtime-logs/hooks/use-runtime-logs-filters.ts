"use client";

import { parseAsArrayOf, parseAsInteger, parseAsString, useQueryStates } from "nuqs";
import { useMemo } from "react";
import { getSeverityColorClass } from "../utils";
import type { RuntimeLogsFilter } from "../types";

const parseAsFilterValueArray = parseAsArrayOf(parseAsString, ",").withDefault([]);

export function useRuntimeLogsFilters() {
  const [queryParams, setQueryParams] = useQueryStates({
    severity: parseAsFilterValueArray,
    searchText: parseAsString.withDefault(""),
    startTime: parseAsInteger,
    endTime: parseAsInteger,
    since: parseAsString.withDefault("6h"),
  });

  const filters = useMemo(() => {
    const result: RuntimeLogsFilter[] = [];

    // Severity filters
    for (const value of queryParams.severity) {
      result.push({
        id: `severity-${value}`,
        field: "severity",
        operator: "is",
        value,
        metadata: {
          label: value,
          colorClass: getSeverityColorClass(value),
        },
      });
    }
    // Search text filter
    if (queryParams.searchText) {
      result.push({
        id: "searchText",
        field: "searchText",
        operator: "is",
        value: queryParams.searchText,
      });
    }

    // Time range filters
    if (queryParams.startTime) {
      result.push({
        id: "startTime",
        field: "startTime",
        operator: "is",
        value: queryParams.startTime,
      });
    }

    if (queryParams.endTime) {
      result.push({
        id: "endTime",
        field: "endTime",
        operator: "is",
        value: queryParams.endTime,
      });
    }

    if (queryParams.since && queryParams.since !== "") {
      result.push({
        id: "since",
        field: "since",
        operator: "is",
        value: queryParams.since,
      });
    }

    return result;
  }, [queryParams]);

  const removeFilter = (filterId: string) => {
    const filter = filters.find((f) => f.id === filterId);
    if (!filter) return;

    const updates: Record<string, unknown> = {};

    if (filter.field === "severity") {
      updates.severity = queryParams.severity.filter((v) => v !== filter.value);
    } else if (filter.field === "searchText") {
      updates.searchText = "";
    } else if (filter.field === "startTime") {
      updates.startTime = null;
    } else if (filter.field === "endTime") {
      updates.endTime = null;
    } else if (filter.field === "since") {
      updates.since = "6h"; // Reset to default
    }

    setQueryParams(updates);
  };

  const updateFiltersFromParams = (newFilters: Partial<typeof queryParams>) => {
    setQueryParams(newFilters);
  };

  // For ControlCloud compatibility - converts filter array to params
  const updateFiltersFromArray = (newFilters: RuntimeLogsFilter[]) => {
    const updates: Partial<typeof queryParams> = {
      severity: newFilters.filter((f) => f.field === "severity").map((f) => String(f.value)),
      searchText: newFilters.find((f) => f.field === "searchText")?.value as string | undefined || "",
      startTime: Number(newFilters.find((f) => f.field === "startTime")?.value) || undefined,
      endTime: Number(newFilters.find((f) => f.field === "endTime")?.value) || undefined,
      since: String(newFilters.find((f) => f.field === "since")?.value) || "6h",
    };
    setQueryParams(updates);
  };

  return {
    filters,
    queryParams,
    removeFilter,
    updateFilters: updateFiltersFromParams,
    updateFiltersFromArray,
  };
}
