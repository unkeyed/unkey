import { parseAsInteger, useQueryStates } from "nuqs";
import { useCallback, useMemo } from "react";
import {
  parseAsFilterValueArray,
  parseAsRelativeTime,
} from "@/components/logs/validation/utils/nuqs-parsers";
import {
  type RequestLogsFilterField,
  type RequestLogsFilterOperator,
  type RequestLogsFilterUrlValue,
  type RequestLogsFilterValue,
  type RequestLogsQuerySearchParams,
  requestLogsFilterFieldConfig,
} from "@/lib/schemas/request-logs.filter.schema";

// Constants
const parseAsFilterValArray = parseAsFilterValueArray<RequestLogsFilterOperator>([
  "is",
  "contains",
  "startsWith",
]);

const arrayFields = [
  "status",
  "methods",
  "paths",
  "appId",
  "deploymentId",
  "environmentId",
  "host",
  "requestId",
  "region",
] as const;
const timeFields = ["startTime", "endTime", "since"] as const;

// Query params configuration
export const queryParamsPayload = {
  status: parseAsFilterValArray,
  methods: parseAsFilterValArray,
  paths: parseAsFilterValArray,
  appId: parseAsFilterValArray,
  deploymentId: parseAsFilterValArray,
  environmentId: parseAsFilterValArray,
  host: parseAsFilterValArray,
  requestId: parseAsFilterValArray,
  region: parseAsFilterValArray,
  startTime: parseAsInteger,
  endTime: parseAsInteger,
  since: parseAsRelativeTime,
} as const;

export const useRequestLogsFilters = () => {
  const [searchParams, setSearchParams] = useQueryStates(queryParamsPayload, {
    history: "push",
  });

  const filters = useMemo(() => {
    const activeFilters: RequestLogsFilterValue[] = [];

    // Handle array filters
    arrayFields.forEach((field) => {
      searchParams[field]?.forEach((item) => {
        if (
          !requestLogsFilterFieldConfig[field].operators.some(
            (operator) => operator === item.operator,
          )
        ) {
          return;
        }
        const colorClass =
          field === "status"
            ? requestLogsFilterFieldConfig.status.getColorClass?.(Number(item.value))
            : undefined;
        activeFilters.push({
          id: crypto.randomUUID(),
          field,
          operator: item.operator,
          value: item.value,
          metadata: colorClass ? { colorClass } : undefined,
        });
      });
    });

    // Handle time filters
    timeFields.forEach((field) => {
      const value = searchParams[field];
      if (value !== null && value !== undefined) {
        activeFilters.push({
          id: crypto.randomUUID(),
          field: field as RequestLogsFilterField,
          operator: "is",
          value: value as string | number,
        });
      }
    });

    return activeFilters;
  }, [searchParams]);

  const updateFilters = useCallback(
    (newFilters: RequestLogsFilterValue[]) => {
      const newParams: Partial<RequestLogsQuerySearchParams> = Object.fromEntries([
        ...arrayFields.map((field) => [field, null]),
        ...timeFields.map((field) => [field, null]),
      ]);

      const filterGroups = arrayFields.reduce(
        (acc, field) => {
          acc[field] = [];
          return acc;
        },
        {} as Record<(typeof arrayFields)[number], RequestLogsFilterUrlValue[]>,
      );

      newFilters.forEach((filter) => {
        if (arrayFields.includes(filter.field as (typeof arrayFields)[number])) {
          filterGroups[filter.field as (typeof arrayFields)[number]].push({
            value: filter.value as string,
            operator: filter.operator,
          });
        } else if (filter.field === "startTime" || filter.field === "endTime") {
          newParams[filter.field] = filter.value as number;
        } else if (filter.field === "since") {
          newParams.since = filter.value as string;
        }
      });

      // Set array filters
      arrayFields.forEach((field) => {
        if (filterGroups[field] !== undefined) {
          (newParams as Record<string, unknown>)[field] =
            filterGroups[field].length > 0 ? filterGroups[field] : null;
        }
      });

      setSearchParams(newParams);
    },
    [setSearchParams],
  );

  const addFilter = useCallback(
    (filter: Omit<RequestLogsFilterValue, "id">) => {
      const newFilter: RequestLogsFilterValue = {
        ...filter,
        id: crypto.randomUUID(),
      };
      updateFilters([...filters, newFilter]);
    },
    [filters, updateFilters],
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
    addFilter,
    removeFilter,
    updateFilters,
  };
};
