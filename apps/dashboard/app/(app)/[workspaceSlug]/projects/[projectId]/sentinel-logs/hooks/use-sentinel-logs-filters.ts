import {
  parseAsFilterValueArray,
  parseAsRelativeTime,
} from "@/components/logs/validation/utils/nuqs-parsers";
import {
  type LogsFilterField as SentinelLogsFilterField,
  type LogsFilterOperator as SentinelLogsFilterOperator,
  type LogsFilterUrlValue as SentinelLogsFilterUrlValue,
  type LogsFilterValue as SentinelLogsFilterValue,
  type QuerySearchParams as SentinelLogsQuerySearchParams,
  logsFilterFieldConfig as sentinelLogsFilterFieldConfig,
} from "@/lib/schemas/logs.filter.schema";
import { parseAsInteger, useQueryStates } from "nuqs";
import { useCallback, useMemo } from "react";

// Constants
const parseAsFilterValArray = parseAsFilterValueArray<SentinelLogsFilterOperator>([
  "is",
  "contains",
  "startsWith",
  "endsWith",
]);

const arrayFields = ["status", "methods", "paths", "host", "requestId"] as const;
const timeFields = ["startTime", "endTime", "since"] as const;

// Query params configuration
export const queryParamsPayload = {
  status: parseAsFilterValArray,
  methods: parseAsFilterValArray,
  paths: parseAsFilterValArray,
  host: parseAsFilterValArray,
  requestId: parseAsFilterValArray,
  startTime: parseAsInteger,
  endTime: parseAsInteger,
  since: parseAsRelativeTime,
} as const;

export const useSentinelLogsFilters = () => {
  const [searchParams, setSearchParams] = useQueryStates(queryParamsPayload, {
    history: "push",
  });

  const filters = useMemo(() => {
    const activeFilters: SentinelLogsFilterValue[] = [];

    // Handle array filters
    arrayFields.forEach((field) => {
      searchParams[field]?.forEach((item) => {
        activeFilters.push({
          id: crypto.randomUUID(),
          field,
          operator: item.operator,
          value: item.value,
          metadata: sentinelLogsFilterFieldConfig[field].getColorClass
            ? {
                colorClass: sentinelLogsFilterFieldConfig[field].getColorClass(
                  //TODO: Handle this later
                  //@ts-expect-error will fix it
                  field === "status" ? Number(item.value) : item.value,
                ),
              }
            : undefined,
        });
      });
    });

    // Handle time filters
    timeFields.forEach((field) => {
      const value = searchParams[field];
      if (value !== null && value !== undefined) {
        activeFilters.push({
          id: crypto.randomUUID(),
          field: field as SentinelLogsFilterField,
          operator: "is",
          value: value as string | number,
        });
      }
    });

    return activeFilters;
  }, [searchParams]);

  const updateFilters = useCallback(
    (newFilters: SentinelLogsFilterValue[]) => {
      const newParams: Partial<SentinelLogsQuerySearchParams> = Object.fromEntries([
        ...arrayFields.map((field) => [field, null]),
        ...timeFields.map((field) => [field, null]),
      ]);

      const filterGroups = arrayFields.reduce(
        (acc, field) => {
          acc[field] = [];
          return acc;
        },
        {} as Record<(typeof arrayFields)[number], SentinelLogsFilterUrlValue[]>,
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
