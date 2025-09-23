import { parseAsRelativeTime } from "@/components/logs/validation/utils/nuqs-parsers";
import { parseAsInteger, useQueryStates } from "nuqs";
import { useCallback, useMemo } from "react";
import {
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

export const useNamespaceListFilters = () => {
  const [searchParams, setSearchParams] = useQueryStates(queryParamsPayload, {
    history: "push",
    throttleMs: 50,
  });

  const filters = useMemo(() => {
    const activeFilters: NamespaceListFilterValue[] = [];

    // Handle array filters
    searchParams.query?.forEach((queryFilter) => {
      activeFilters.push({
        id: crypto.randomUUID(),
        field: "query",
        operator: queryFilter.operator,
        value: queryFilter.value,
      });
    });

    return activeFilters;
  }, [searchParams]);

  const updateFilters = useCallback(
    (newFilters: NamespaceListFilterValue[]) => {
      const newParams: Partial<NamespaceListQuerySearchParams> = Object.fromEntries([
        ...arrayFields.map((field) => [field, null]),
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
