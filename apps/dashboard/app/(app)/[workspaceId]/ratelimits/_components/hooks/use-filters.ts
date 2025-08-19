import {
  parseAsFilterValueArray,
  parseAsRelativeTime,
} from "@/components/logs/validation/utils/nuqs-parsers";
import { parseAsInteger, useQueryStates } from "nuqs";
import { useCallback, useMemo } from "react";
import type {
  RatelimitListFilterField,
  RatelimitListFilterOperator,
  RatelimitListFilterValue,
  RatelimitQuerySearchParams,
} from "../filters.schema";

const parseAsFilterValArray = parseAsFilterValueArray<RatelimitListFilterOperator>([
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
    const activeFilters: RatelimitListFilterValue[] = [];

    ["startTime", "endTime", "since"].forEach((field) => {
      const value = searchParams[field as keyof RatelimitQuerySearchParams];
      if (value !== null && value !== undefined) {
        activeFilters.push({
          id: crypto.randomUUID(),
          field: field as RatelimitListFilterField,
          operator: "is",
          value: value as string | number,
        });
      }
    });

    return activeFilters;
  }, [searchParams]);

  const updateFilters = useCallback(
    (newFilters: RatelimitListFilterValue[]) => {
      const newParams: Partial<RatelimitQuerySearchParams> = {
        startTime: null,
        endTime: null,
        since: null,
      };

      newFilters.forEach((filter) => {
        switch (filter.field) {
          case "startTime":
          case "endTime":
            newParams[filter.field] = filter.value as number;
            break;
          case "since":
            newParams.since = filter.value as string;
            break;
        }
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
};
