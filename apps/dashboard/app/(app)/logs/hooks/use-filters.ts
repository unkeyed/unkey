import { useQueryStates } from "nuqs";
import { useCallback, useMemo } from "react";
import {
  type LogsFilterField,
  type LogsFilterValue,
  type QuerySearchParams,
  logsFilterFieldConfig,
  queryParamsPayload,
} from "../filters.schema";

export const useFilters = () => {
  const [searchParams, setSearchParams] = useQueryStates(queryParamsPayload, {
    history: "push",
  });

  const filters = useMemo(() => {
    const activeFilters: LogsFilterValue[] = [];

    Object.entries(searchParams).forEach(([fieldName, fieldValue]) => {
      const field = fieldName as LogsFilterField;
      const config = logsFilterFieldConfig[field];

      if (!config || fieldValue === null || fieldValue === undefined) {
        return;
      }

      // Handle time fields (direct values)
      if ("isTimeField" in config && config.isTimeField) {
        activeFilters.push({
          id: crypto.randomUUID(),
          field,
          operator: "is",
          value: fieldValue as number,
        });
        return;
      }

      // Handle relative time fields (direct values)
      if ("isRelativeTimeField" in config && config.isRelativeTimeField) {
        activeFilters.push({
          id: crypto.randomUUID(),
          field,
          operator: "is",
          value: fieldValue as string,
        });
        return;
      }

      // Handle regular filter arrays
      if (Array.isArray(fieldValue)) {
        fieldValue.forEach((filterItem) => {
          activeFilters.push({
            id: crypto.randomUUID(),
            field,
            operator: filterItem.operator,
            value: filterItem.value,
            metadata:
              "getColorClass" in config
                ? {
                    colorClass: config.getColorClass(filterItem.value),
                  }
                : undefined,
          });
        });
      }
    });

    return activeFilters;
  }, [searchParams]);

  const updateFilters = useCallback(
    (newFilters: LogsFilterValue[]) => {
      const newParams = {} as Record<LogsFilterField, unknown>;
      Object.keys(logsFilterFieldConfig).forEach((field) => {
        newParams[field as LogsFilterField] = null;
      });
      const filterGroups: Record<LogsFilterField, LogsFilterValue[]> =
        {} as Record<LogsFilterField, LogsFilterValue[]>;

      newFilters.forEach((filter) => {
        const field = filter.field as LogsFilterField;
        if (!filterGroups[field]) {
          filterGroups[field] = [];
        }
        filterGroups[field].push(filter);
      });

      (
        Object.entries(filterGroups) as [LogsFilterField, LogsFilterValue[]][]
      ).forEach(([field, filters]) => {
        const config = logsFilterFieldConfig[field];

        if ("isTimeField" in config && config.isTimeField) {
          const timeFilter = filters[0];
          if (timeFilter) {
            newParams[field] = timeFilter.value as number;
          }
          return;
        }

        if ("isRelativeTimeField" in config && config.isRelativeTimeField) {
          const relativeTimeFilter = filters[0];
          if (relativeTimeFilter) {
            newParams[field] = relativeTimeFilter.value as string;
          }
          return;
        }

        const filterArray = filters.map((filter) => ({
          operator: filter.operator,
          value: filter.value,
        }));

        newParams[field] = filterArray.length > 0 ? filterArray : null;
      });

      setSearchParams(newParams as Partial<QuerySearchParams>);
    },
    [setSearchParams]
  );

  const removeFilter = useCallback(
    (id: string) => {
      const newFilters = filters.filter((f) => f.id !== id);
      updateFilters(newFilters);
    },
    [filters, updateFilters]
  );

  const addFilter = useCallback(
    (filter: Omit<LogsFilterValue, "id">) => {
      const newFilter: LogsFilterValue = {
        ...filter,
        id: crypto.randomUUID(),
      };
      updateFilters([...filters, newFilter]);
    },
    [filters, updateFilters]
  );

  const clearAllFilters = useCallback(() => {
    updateFilters([]);
  }, [updateFilters]);

  return {
    filters,
    searchParams,
    removeFilter,
    addFilter,
    updateFilters,
    clearAllFilters,
  };
};
