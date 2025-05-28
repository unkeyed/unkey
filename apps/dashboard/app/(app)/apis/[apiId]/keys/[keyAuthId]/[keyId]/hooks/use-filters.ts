import {
  parseAsFilterValueArray,
  parseAsRelativeTime,
} from "@/components/logs/validation/utils/nuqs-parsers";
import { parseAsInteger, useQueryStates } from "nuqs";
import { useCallback, useMemo } from "react";
import {
  type IsOnlyUrlValue,
  type KeyDetailsFilterField,
  type KeyDetailsFilterValue,
  type KeysQuerySearchParams,
  keyDetailsFilterFieldConfig,
} from "../filters.schema";

const parseAsIsOnlyFilterArray = parseAsFilterValueArray<"is">(["is"]);

export const queryParamsPayload = {
  startTime: parseAsInteger,
  endTime: parseAsInteger,
  since: parseAsRelativeTime,
  outcomes: parseAsIsOnlyFilterArray,
} as const;

export const useFilters = () => {
  const [searchParams, setSearchParams] = useQueryStates(queryParamsPayload, {
    history: "push",
  });

  const filters = useMemo(() => {
    const activeFilters: KeyDetailsFilterValue[] = [];

    for (const [field, value] of Object.entries(searchParams)) {
      if (!Array.isArray(value) || field !== "outcomes") {
        continue;
      }

      for (const filterItem of value) {
        const baseFilter = {
          id: crypto.randomUUID(),
          field: field as KeyDetailsFilterField,
          operator: filterItem.operator,
          value: filterItem.value,
        };

        if (field === "outcomes") {
          activeFilters.push({
            ...baseFilter,
            metadata: {
              colorClass: keyDetailsFilterFieldConfig.outcomes.getColorClass?.(
                filterItem.value as string,
              ),
            },
          });
        } else {
          activeFilters.push(baseFilter);
        }
      }
    }

    ["startTime", "endTime", "since"].forEach((field) => {
      const value = searchParams[field as keyof KeysQuerySearchParams];
      if (value !== null && value !== undefined) {
        activeFilters.push({
          id: crypto.randomUUID(),
          field: field as KeyDetailsFilterField,
          operator: "is",
          value: value as string | number,
        });
      }
    });

    return activeFilters;
  }, [searchParams]);

  const updateFilters = useCallback(
    (newFilters: KeyDetailsFilterValue[]) => {
      const newParams: Partial<KeysQuerySearchParams> = {
        startTime: null,
        endTime: null,
        since: null,
        outcomes: null,
      };

      const outcomeFilters: IsOnlyUrlValue[] = [];

      newFilters.forEach((filter) => {
        const fieldConfig = keyDetailsFilterFieldConfig[filter.field];
        const validOperators = fieldConfig.operators;

        const operator = validOperators.includes(filter.operator)
          ? filter.operator
          : validOperators[0];
        if (operator !== "is") {
          throw new Error("Invalid filter operator. Only 'is' operator is allowed.");
        }

        switch (filter.field) {
          case "outcomes":
            if (typeof filter.value === "string") {
              outcomeFilters.push({
                value: filter.value,
                operator: "is",
              });
            }
            break;

          case "startTime":
          case "endTime": {
            const numValue =
              typeof filter.value === "number"
                ? filter.value
                : typeof filter.value === "string"
                  ? Number(filter.value)
                  : Number.NaN;

            if (!Number.isNaN(numValue)) {
              newParams[filter.field] = numValue;
            }
            break;
          }

          case "since":
            if (typeof filter.value === "string") {
              newParams.since = filter.value;
            }
            break;
        }
      });

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
