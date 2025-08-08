import {
  parseAsFilterValueArray,
  parseAsRelativeTime,
} from "@/components/logs/validation/utils/nuqs-parsers";
import { parseAsInteger, useQueryStates } from "nuqs";
import { useCallback, useMemo } from "react";
import {
  type RatelimitOverviewFilterField,
  type RatelimitOverviewFilterOperator,
  type RatelimitOverviewFilterUrlValue,
  type RatelimitOverviewFilterValue,
  type RatelimitQuerySearchParams,
  ratelimitOverviewFilterFieldConfig,
} from "../filters.schema";

const parseAsFilterValArray = parseAsFilterValueArray<RatelimitOverviewFilterOperator>([
  "is",
  "contains",
]);
export const queryParamsPayload = {
  identifiers: parseAsFilterValArray,
  startTime: parseAsInteger,
  endTime: parseAsInteger,
  since: parseAsRelativeTime,
  status: parseAsFilterValArray,
} as const;

export const useFilters = () => {
  const [searchParams, setSearchParams] = useQueryStates(queryParamsPayload, {
    history: "push",
  });
  const filters = useMemo(() => {
    const activeFilters: RatelimitOverviewFilterValue[] = [];

    searchParams.identifiers?.forEach((pathFilter) => {
      activeFilters.push({
        id: crypto.randomUUID(),
        field: "identifiers",
        operator: pathFilter.operator,
        value: pathFilter.value,
      });
    });

    searchParams.status?.forEach((statusFilter) => {
      activeFilters.push({
        id: crypto.randomUUID(),
        field: "status",
        operator: statusFilter.operator,
        value: statusFilter.value,
        metadata: {
          colorClass: ratelimitOverviewFilterFieldConfig.status.getColorClass?.(
            statusFilter.value as string,
          ),
        },
      });
    });

    ["startTime", "endTime", "since"].forEach((field) => {
      const value = searchParams[field as keyof RatelimitQuerySearchParams];
      if (value !== null && value !== undefined) {
        activeFilters.push({
          id: crypto.randomUUID(),
          field: field as RatelimitOverviewFilterField,
          operator: "is",
          value: value as string | number,
        });
      }
    });

    return activeFilters;
  }, [searchParams]);

  const updateFilters = useCallback(
    (newFilters: RatelimitOverviewFilterValue[]) => {
      const newParams: Partial<RatelimitQuerySearchParams> = {
        startTime: null,
        endTime: null,
        since: null,
        identifiers: null,
      };

      const identifierFilters: RatelimitOverviewFilterUrlValue[] = [];
      const statusFilters: RatelimitOverviewFilterUrlValue[] = [];

      newFilters.forEach((filter) => {
        switch (filter.field) {
          case "identifiers":
            identifierFilters.push({
              value: filter.value,
              operator: filter.operator,
            });
            break;

          case "status":
            statusFilters.push({
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
      newParams.identifiers = identifierFilters.length > 0 ? identifierFilters : null;
      newParams.status = statusFilters.length > 0 ? statusFilters : null;
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
};
