import { type Parser, parseAsInteger, useQueryStates } from "nuqs";
import { useCallback, useMemo } from "react";
import { filterFieldConfig } from "../filters.schema";
import type {
  FilterField,
  FilterOperator,
  FilterUrlValue,
  FilterValue,
  QuerySearchParams,
} from "../filters.type";

export const parseAsRelativeTime: Parser<string | null> = {
  parse: (str: string | null) => {
    if (!str) {
      return null;
    }

    try {
      // Validate the format matches one or more of: number + (h|d|m)
      const isValid = /^(\d+[hdm])+$/.test(str);
      if (!isValid) {
        return null;
      }
      return str;
    } catch {
      return null;
    }
  },
  serialize: (value: string | null) => {
    if (!value) {
      return "";
    }
    return value;
  },
};

export const parseAsFilterValueArray: Parser<FilterUrlValue[]> = {
  parse: (str: string | null) => {
    if (!str) {
      return [];
    }
    try {
      // Format: operator:value,operator:value (e.g., "is:200,is:404")
      return str.split(",").map((item) => {
        const [operator, val] = item.split(/:(.+)/);
        if (!["is", "contains", "startsWith", "endsWith"].includes(operator)) {
          throw new Error("Invalid operator");
        }
        return {
          operator: operator as FilterOperator,
          value: val,
        };
      });
    } catch {
      return [];
    }
  },
  serialize: (value: FilterUrlValue[]) => {
    if (!value?.length) {
      return "";
    }
    return value.map((v) => `${v.operator}:${v.value}`).join(",");
  },
};

export const queryParamsPayload = {
  requestIds: parseAsFilterValueArray,
  identifiers: parseAsFilterValueArray,
  startTime: parseAsInteger,
  endTime: parseAsInteger,
  status: parseAsFilterValueArray,
  since: parseAsRelativeTime,
} as const;

export const useFilters = () => {
  const [searchParams, setSearchParams] = useQueryStates(queryParamsPayload);

  const filters = useMemo(() => {
    const activeFilters: FilterValue[] = [];

    searchParams.requestIds?.forEach((requestIdFilter) => {
      activeFilters.push({
        id: crypto.randomUUID(),
        field: "requestIds",
        operator: requestIdFilter.operator,
        value: requestIdFilter.value,
      });
    });

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
          colorClass: filterFieldConfig.status.getColorClass?.(statusFilter.value as string),
        },
      });
    });

    ["startTime", "endTime", "since"].forEach((field) => {
      const value = searchParams[field as keyof QuerySearchParams];
      if (value !== null && value !== undefined) {
        activeFilters.push({
          id: crypto.randomUUID(),
          field: field as FilterField,
          operator: "is",
          value: value as string | number,
        });
      }
    });

    return activeFilters;
  }, [searchParams]);

  const updateFilters = useCallback(
    (newFilters: FilterValue[]) => {
      const newParams: Partial<QuerySearchParams> = {
        requestIds: null,
        startTime: null,
        endTime: null,
        since: null,
        identifiers: null,
        status: null,
      };

      // Group filters by field
      const requestIdFilters: FilterUrlValue[] = [];
      const statusFilters: FilterUrlValue[] = [];
      const identifierFilters: FilterUrlValue[] = [];

      newFilters.forEach((filter) => {
        switch (filter.field) {
          case "requestIds":
            requestIdFilters.push({
              value: filter.value as string,
              operator: filter.operator,
            });
            break;
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
      newParams.requestIds = requestIdFilters.length > 0 ? requestIdFilters : null;
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
