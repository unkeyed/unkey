import {
  parseAsFilterValueArray,
  parseAsRelativeTime,
} from "@/components/logs/validation/utils/nuqs-parsers";
import { parseAsInteger, useQueryStates } from "nuqs";
import { useCallback, useMemo } from "react";
import {
  type KeysOverviewFilterField,
  type KeysOverviewFilterOperator,
  type KeysOverviewFilterUrlValue,
  type KeysOverviewFilterValue,
  type KeysQuerySearchParams,
  keysOverviewFilterFieldConfig,
} from "../filters.schema";

const parseAsFilterValArray = parseAsFilterValueArray<KeysOverviewFilterOperator>([
  "is",
  "contains",
]);

export const queryParamsPayload = {
  keyIds: parseAsFilterValArray,
  startTime: parseAsInteger,
  endTime: parseAsInteger,
  since: parseAsRelativeTime,
  outcomes: parseAsFilterValArray,
} as const;

export const useFilters = () => {
  const [searchParams, setSearchParams] = useQueryStates(queryParamsPayload);

  const filters = useMemo(() => {
    const activeFilters: KeysOverviewFilterValue[] = [];

    searchParams.keyIds?.forEach((keyFilter) => {
      activeFilters.push({
        id: crypto.randomUUID(),
        field: "keyIds",
        operator: keyFilter.operator,
        value: keyFilter.value,
      });
    });

    searchParams.outcomes?.forEach((outcomeFilter) => {
      activeFilters.push({
        id: crypto.randomUUID(),
        field: "outcomes",
        operator: outcomeFilter.operator,
        value: outcomeFilter.value,
        metadata: {
          colorClass: keysOverviewFilterFieldConfig.outcomes.getColorClass?.(
            outcomeFilter.value as string,
          ),
        },
      });
    });

    ["startTime", "endTime", "since"].forEach((field) => {
      const value = searchParams[field as keyof KeysQuerySearchParams];
      if (value !== null && value !== undefined) {
        activeFilters.push({
          id: crypto.randomUUID(),
          field: field as KeysOverviewFilterField,
          operator: "is",
          value: value as string | number,
        });
      }
    });

    return activeFilters;
  }, [searchParams]);

  const updateFilters = useCallback(
    (newFilters: KeysOverviewFilterValue[]) => {
      const newParams: Partial<KeysQuerySearchParams> = {
        startTime: null,
        endTime: null,
        since: null,
        keyIds: null,
        outcomes: null,
      };

      const keyIdFilters: KeysOverviewFilterUrlValue[] = [];
      const outcomeFilters: KeysOverviewFilterUrlValue[] = [];

      newFilters.forEach((filter) => {
        switch (filter.field) {
          case "keyIds":
            keyIdFilters.push({
              value: filter.value,
              operator: filter.operator,
            });
            break;
          case "outcomes":
            outcomeFilters.push({
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
      newParams.keyIds = keyIdFilters.length > 0 ? keyIdFilters : null;
      newParams.outcomes = outcomeFilters.length > 0 ? outcomeFilters : null;

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
