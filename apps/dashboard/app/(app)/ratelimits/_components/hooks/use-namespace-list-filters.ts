import { parseAsRelativeTime } from "@/components/logs/validation/utils/nuqs-parsers";
import { parseAsInteger, useQueryStates } from "nuqs";
import { useCallback, useMemo } from "react";
import {
  type NamespaceListFilterField,
  type NamespaceListFilterUrlValue,
  type NamespaceListFilterValue,
  type NamespaceListQuerySearchParams,
  parseAsAllOperatorsFilterArray,
} from "../namespace-list-filters.schema";

export const queryParamsPayload = {
  query: parseAsAllOperatorsFilterArray,
  startTime: parseAsInteger,
  endTime: parseAsInteger,
  since: parseAsRelativeTime,
} as const;

const arrayFields = ["query"] as const;
const timeFields = ["startTime", "endTime", "since"] as const;

export const useNamespaceListFilters = () => {
  const [searchParams, setSearchParams] = useQueryStates(queryParamsPayload, {
    history: "push",
  });

  const filters = useMemo(() => {
    const activeFilters: NamespaceListFilterValue[] = [];

    // Handle array filters
    searchParams.query?.forEach((status) => {
      activeFilters.push({
        id: crypto.randomUUID(),
        field: "query",
        operator: status.operator,
        value: status.value,
      });
    });

    // Handle time filters
    ["startTime", "endTime", "since"].forEach((field) => {
      const value = searchParams[field as keyof NamespaceListQuerySearchParams];
      if (value !== null && value !== undefined) {
        activeFilters.push({
          id: crypto.randomUUID(),
          field: field as NamespaceListFilterField,
          operator: "is",
          value: value as string | number,
        });
      }
    });

    return activeFilters;
  }, [searchParams]);

  const updateFilters = useCallback(
    (newFilters: NamespaceListFilterValue[]) => {
      const newParams: Partial<NamespaceListQuerySearchParams> = Object.fromEntries([
        ...arrayFields.map((field) => [field, null]),
        ...timeFields.map((field) => [field, null]),
      ]);

      const filterGroups = arrayFields.reduce(
        (acc, field) => {
          acc[field] = [];
          return acc;
        },
        {} as Record<(typeof arrayFields)[number], NamespaceListFilterUrlValue[]>,
      );

      newFilters.forEach((filter) => {
        // This will handle any array like filter
        if (arrayFields.includes(filter.field as (typeof arrayFields)[number])) {
          filterGroups[filter.field as (typeof arrayFields)[number]].push({
            value: filter.value as string,
            operator: filter.operator,
          });
        } else {
          switch (filter.field) {
            case "startTime":
            case "endTime":
              if (typeof filter.value === "number") {
                newParams[filter.field] = filter.value;
              }
              break;
            case "since":
              if (typeof filter.value === "string") {
                newParams.since = filter.value;
              }
              break;
          }
        }
      });

      // Set array filters
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
};
