import {
  parseAsFilterValueArray,
  parseAsRelativeTime,
} from "@/components/logs/validation/utils/nuqs-parsers";
import { parseAsInteger, useQueryStates } from "nuqs";
import { useCallback, useMemo } from "react";
import {
  type AllOperatorsUrlValue,
  type IsContainsUrlValue,
  type IsOnlyUrlValue,
  type KeysOverviewFilterField,
  type KeysOverviewFilterOperator,
  type KeysOverviewFilterValue,
  type KeysQuerySearchParams,
  keysOverviewFilterFieldConfig,
} from "../filters.schema";

const parseAsAllOperatorsFilterArray = parseAsFilterValueArray<KeysOverviewFilterOperator>([
  "is",
  "contains",
  "startsWith",
  "endsWith",
]);

const parseAsIsContainsFilterArray = parseAsFilterValueArray<"is" | "contains">(["is", "contains"]);

const parseAsIsOnlyFilterArray = parseAsFilterValueArray<"is">(["is"]);

// Map fields to appropriate parsers based on their allowed operators in the config
export const queryParamsPayload = {
  keyIds: parseAsIsContainsFilterArray,
  names: parseAsAllOperatorsFilterArray,
  startTime: parseAsInteger,
  endTime: parseAsInteger,
  since: parseAsRelativeTime,
  outcomes: parseAsIsOnlyFilterArray,
  identities: parseAsAllOperatorsFilterArray,
} as const;

export const useFilters = () => {
  const [searchParams, setSearchParams] = useQueryStates(queryParamsPayload);

  const filters = useMemo(() => {
    const activeFilters: KeysOverviewFilterValue[] = [];

    for (const [field, value] of Object.entries(searchParams)) {
      if (!Array.isArray(value) || !["keyIds", "names", "identities", "outcomes"].includes(field)) {
        continue;
      }

      for (const filterItem of value) {
        const baseFilter = {
          id: crypto.randomUUID(),
          field: field as KeysOverviewFilterField,
          operator: filterItem.operator,
          value: filterItem.value,
        };

        if (field === "outcomes") {
          activeFilters.push({
            ...baseFilter,
            metadata: {
              colorClass: keysOverviewFilterFieldConfig.outcomes.getColorClass?.(
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
        names: null,
        identities: null,
        outcomes: null,
      };

      const keyIdFilters: IsContainsUrlValue[] = [];
      const nameFilters: AllOperatorsUrlValue[] = [];
      const identitiesFilters: AllOperatorsUrlValue[] = [];
      const outcomeFilters: IsOnlyUrlValue[] = [];

      newFilters.forEach((filter) => {
        const fieldConfig = keysOverviewFilterFieldConfig[filter.field];
        const validOperators = fieldConfig.operators;

        const operator = validOperators.includes(filter.operator)
          ? filter.operator
          : validOperators[0];

        switch (filter.field) {
          case "keyIds":
            if (typeof filter.value === "string") {
              keyIdFilters.push({
                value: filter.value,
                operator: operator as "is" | "contains",
              });
            }
            break;

          case "names":
          case "identities":
            if (typeof filter.value === "string") {
              const filterValue = {
                value: filter.value,
                operator: operator as "is" | "contains" | "startsWith" | "endsWith",
              };

              filter.field === "names"
                ? nameFilters.push(filterValue)
                : identitiesFilters.push(filterValue);
            }
            break;

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

      newParams.keyIds = keyIdFilters.length > 0 ? keyIdFilters : null;
      newParams.names = nameFilters.length > 0 ? nameFilters : null;
      newParams.identities = identitiesFilters.length > 0 ? identitiesFilters : null;
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
