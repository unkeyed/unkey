import {
  parseAsFilterValueArray,
  parseAsRelativeTime,
} from "@/components/logs/validation/utils/nuqs-parsers";
import type { Filter } from "@/components/logs/verification-chart";
import { parseAsInteger, useQueryStates } from "nuqs";
import { useCallback, useMemo } from "react";
import {
  type AllOperatorsUrlValue,
  type IsOnlyUrlValue,
  type KeysQuerySearchParams,
  keyDetailsFilterFieldConfig,
} from "../filters.schema";

const parseAsIsOnlyFilterArray = parseAsFilterValueArray<"is">(["is"]);
const parseAsAllOperatorsFilterArray = parseAsFilterValueArray<
  "is" | "contains" | "startsWith" | "endsWith"
>(["is", "contains", "startsWith", "endsWith"]);

export const queryParamsPayload = {
  startTime: parseAsInteger,
  endTime: parseAsInteger,
  since: parseAsRelativeTime,
  tags: parseAsAllOperatorsFilterArray,
  outcomes: parseAsIsOnlyFilterArray,
} as const;

export const useFilters = () => {
  const [searchParams, setSearchParams] = useQueryStates(queryParamsPayload, {
    history: "push",
  });

  const filters = useMemo(() => {
    const activeFilters: Filter[] = [];

    for (const [field, value] of Object.entries(searchParams)) {
      if (!Array.isArray(value) || (field !== "outcomes" && field !== "tags")) {
        continue;
      }

      for (const filterItem of value) {
        activeFilters.push({
          id: crypto.randomUUID(),
          field: field,
          operator: filterItem.operator,
          value: filterItem.value,
        });
      }
    }

    ["startTime", "endTime", "since"].forEach((field) => {
      const value = searchParams[field as keyof KeysQuerySearchParams];
      if (value !== null && value !== undefined) {
        activeFilters.push({
          id: crypto.randomUUID(),
          field: field,
          operator: "is",
          value: value as string | number,
        });
      }
    });

    return activeFilters;
  }, [searchParams]);

  const updateFilters = useCallback(
    (newFilters: Filter[]) => {
      const newParams: Partial<KeysQuerySearchParams> = {
        startTime: null,
        endTime: null,
        since: null,
        tags: null,
        outcomes: null,
      };

      const outcomeFilters: IsOnlyUrlValue[] = [];
      const tagFilters: AllOperatorsUrlValue[] = [];

      newFilters.forEach((filter) => {
        const fieldConfig = keyDetailsFilterFieldConfig[filter.field as keyof typeof keyDetailsFilterFieldConfig];
        if (!fieldConfig) return;
        
        const validOperators = fieldConfig.operators;
        const operator = validOperators.includes(filter.operator as any)
          ? filter.operator
          : validOperators[0];
          
        switch (filter.field) {
          case "tags":
            if (!validOperators.includes(filter.operator as any)) {
              throw new Error(
                `Invalid filter operator for tags. Allowed operators are: ${validOperators.join(
                  ", ",
                )}`,
              );
            }
            if (typeof filter.value === "string") {
              tagFilters.push({
                value: filter.value,
                operator: filter.operator as "is" | "contains" | "startsWith" | "endsWith",
              });
            }
            break;

          case "outcomes":
            if (operator !== "is") {
              throw new Error(
                "Invalid filter operator for outcomes. Only 'is' operator is allowed.",
              );
            }
            if (typeof filter.value === "string") {
              outcomeFilters.push({
                value: filter.value,
                operator: "is",
              });
            }
            break;

          case "startTime":
          case "endTime": {
            if (operator !== "is") {
              throw new Error(
                "Invalid filter operator for time fields. Only 'is' operator is allowed.",
              );
            }
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
            if (operator !== "is") {
              throw new Error(
                "Invalid filter operator for since field. Only 'is' operator is allowed.",
              );
            }
            if (typeof filter.value === "string") {
              newParams.since = filter.value;
            }
            break;
        }
      });

      newParams.tags = tagFilters.length > 0 ? tagFilters : null;
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
