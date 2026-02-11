import {
  parseAsFilterValueArray,
  parseAsRelativeTime,
} from "@/components/logs/validation/utils/nuqs-parsers";
import {
  type LogsFilterField,
  type LogsFilterOperator,
  type LogsFilterUrlValue,
  type LogsFilterValue,
  type QuerySearchParams,
  logsFilterFieldConfig,
} from "@/lib/schemas/logs.filter.schema";
import { parseAsInteger, useQueryStates } from "nuqs";
import { useCallback, useMemo } from "react";

const parseAsFilterValArray = parseAsFilterValueArray<LogsFilterOperator>(["is", "contains"]);
export const queryParamsPayload = {
  requestId: parseAsFilterValArray,
  host: parseAsFilterValArray,
  methods: parseAsFilterValArray,
  paths: parseAsFilterValArray,
  status: parseAsFilterValArray,
  startTime: parseAsInteger,
  endTime: parseAsInteger,
  since: parseAsRelativeTime,
} as const;

export const useFilters = () => {
  const [searchParams, setSearchParams] = useQueryStates(queryParamsPayload, {
    history: "push",
  });

  const filters = useMemo(() => {
    const activeFilters: LogsFilterValue[] = [];

    searchParams.status?.forEach((status) => {
      activeFilters.push({
        id: crypto.randomUUID(),
        field: "status",
        operator: status.operator,
        value: status.value,
        metadata: {
          colorClass: logsFilterFieldConfig.status.getColorClass?.(status.value as number),
        },
      });
    });

    searchParams.methods?.forEach((method) => {
      activeFilters.push({
        id: crypto.randomUUID(),
        field: "methods",
        operator: method.operator,
        value: method.value,
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

    ["startTime", "endTime", "since"].forEach((field) => {
      const value = searchParams[field as keyof QuerySearchParams];
      if (value !== null && value !== undefined) {
        activeFilters.push({
          id: crypto.randomUUID(),
          field: field as LogsFilterField,
          operator: "is",
          value: value as string | number,
        });
      }
    });

    return activeFilters;
  }, [searchParams]);

  const updateFilters = useCallback(
    (newFilters: LogsFilterValue[]) => {
      const newParams: Partial<QuerySearchParams> = {
        paths: null,
        host: null,
        requestId: null,
        startTime: null,
        endTime: null,
        methods: null,
        status: null,
        since: null,
      };

      // Group filters by field
      const responseStatusFilters: LogsFilterUrlValue[] = [];
      const methodFilters: LogsFilterUrlValue[] = [];
      const pathFilters: LogsFilterUrlValue[] = [];
      const hostFilters: LogsFilterUrlValue[] = [];
      const requestIdFilters: LogsFilterUrlValue[] = [];

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
          case "since":
            newParams.since = filter.value as string;
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

  return {
    filters,
    removeFilter,
    updateFilters,
  };
};
