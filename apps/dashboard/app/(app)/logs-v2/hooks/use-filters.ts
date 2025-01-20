import { type Parser, parseAsInteger, useQueryStates } from "nuqs";
import { useCallback, useMemo } from "react";
import { filterFieldConfig } from "../filters.schema";
import type {
  FilterField,
  FilterOperator,
  FilterUrlValue,
  FilterValue,
  HttpMethod,
  QuerySearchParams,
  ResponseStatus,
} from "../filters.type";

const parseAsFilterValueArray: Parser<FilterUrlValue[]> = {
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
  requestId: parseAsFilterValueArray,
  host: parseAsFilterValueArray,
  methods: parseAsFilterValueArray,
  paths: parseAsFilterValueArray,
  status: parseAsFilterValueArray,
  startTime: parseAsInteger,
  endTime: parseAsInteger,
} as const;

export const useFilters = () => {
  const [searchParams, setSearchParams] = useQueryStates(queryParamsPayload);

  const filters = useMemo(() => {
    const activeFilters: FilterValue[] = [];

    searchParams.status?.forEach((status) => {
      activeFilters.push({
        id: crypto.randomUUID(),
        field: "status",
        operator: status.operator,
        value: status.value as ResponseStatus,
        metadata: {
          colorClass: filterFieldConfig.status.getColorClass?.(status.value as number),
        },
      });
    });

    searchParams.methods?.forEach((method) => {
      activeFilters.push({
        id: crypto.randomUUID(),
        field: "methods",
        operator: method.operator,
        value: method.value as HttpMethod,
      });
    });

    searchParams.paths?.forEach((pathFilter) => {
      activeFilters.push({
        id: crypto.randomUUID(),
        field: "paths",
        operator: pathFilter.operator,
        value: pathFilter.value,
      });
    });

    searchParams.host?.forEach((hostFilter) => {
      activeFilters.push({
        id: crypto.randomUUID(),
        field: "host",
        operator: hostFilter.operator,
        value: hostFilter.value,
      });
    });

    searchParams.requestId?.forEach((requestIdFilter) => {
      activeFilters.push({
        id: crypto.randomUUID(),
        field: "requestId",
        operator: requestIdFilter.operator,
        value: requestIdFilter.value,
      });
    });

    ["startTime", "endTime"].forEach((field) => {
      const value = searchParams[field as keyof QuerySearchParams];
      if (value !== null && value !== undefined) {
        activeFilters.push({
          id: crypto.randomUUID(),
          field: field as FilterField,
          operator: "is",
          value: value as number,
        });
      }
    });

    return activeFilters;
  }, [searchParams]);

  const updateFilters = useCallback(
    (newFilters: FilterValue[]) => {
      const newParams: Partial<QuerySearchParams> = {
        paths: null,
        host: null,
        requestId: null,
        startTime: null,
        endTime: null,
        methods: null,
        status: null,
      };

      // Group filters by field
      const responseStatusFilters: FilterUrlValue[] = [];
      const methodFilters: FilterUrlValue[] = [];
      const pathFilters: FilterUrlValue[] = [];
      const hostFilters: FilterUrlValue[] = [];
      const requestIdFilters: FilterUrlValue[] = [];

      newFilters.forEach((filter) => {
        switch (filter.field) {
          case "status":
            responseStatusFilters.push({
              value: filter.value,
              operator: filter.operator,
            });
            break;
          case "methods":
            methodFilters.push({
              value: filter.value,
              operator: filter.operator,
            });
            break;
          case "paths":
            pathFilters.push({
              value: filter.value,
              operator: filter.operator,
            });
            break;
          case "host":
            hostFilters.push({
              value: filter.value as string,
              operator: filter.operator,
            });
            break;
          case "requestId":
            requestIdFilters.push({
              value: filter.value as string,
              operator: filter.operator,
            });
            break;
          case "startTime":
          case "endTime":
            newParams[filter.field] = filter.value as number;
            break;
        }
      });

      // Set arrays to null when empty, otherwise use the filtered values
      newParams.status = responseStatusFilters.length > 0 ? responseStatusFilters : null;
      newParams.methods = methodFilters.length > 0 ? methodFilters : null;
      newParams.paths = pathFilters.length > 0 ? pathFilters : null;
      newParams.host = hostFilters.length > 0 ? hostFilters : null;
      newParams.requestId = requestIdFilters.length > 0 ? requestIdFilters : null;

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

  const addFilter = useCallback(
    (
      field: FilterField,
      operator: FilterOperator,
      value: string | number | ResponseStatus | HttpMethod,
    ) => {
      const newFilter: FilterValue = {
        id: crypto.randomUUID(),
        field,
        operator,
        value,
        metadata:
          field === "status"
            ? {
                colorClass: filterFieldConfig.status.getColorClass?.(value as number),
              }
            : undefined,
      };

      updateFilters([...filters, newFilter]);
    },
    [filters, updateFilters],
  );

  return {
    filters,
    addFilter,
    removeFilter,
    updateFilters,
  };
};
