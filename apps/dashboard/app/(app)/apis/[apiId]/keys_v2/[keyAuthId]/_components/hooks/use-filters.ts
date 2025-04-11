import { parseAsFilterValueArray } from "@/components/logs/validation/utils/nuqs-parsers";

import { useQueryStates } from "nuqs";
import { useCallback, useMemo } from "react";
import {
  type AllOperatorsUrlValue,
  type KeysListFilterField,
  type KeysListFilterOperator,
  type KeysListFilterValue,
  type KeysQuerySearchParams,
  keysListFilterFieldConfig,
} from "../filters.schema";

const parseAsAllOperatorsFilterArray = parseAsFilterValueArray<KeysListFilterOperator>([
  "is",
  "contains",
  "startsWith",
  "endsWith",
]);

export const queryParamsPayload = {
  keyIds: parseAsAllOperatorsFilterArray,
  names: parseAsAllOperatorsFilterArray,
  identities: parseAsAllOperatorsFilterArray,
} as const;

export const useFilters = () => {
  const [searchParams, setSearchParams] = useQueryStates(queryParamsPayload, {
    history: "push",
  });

  const filters = useMemo(() => {
    const activeFilters: KeysListFilterValue[] = [];

    for (const [field, value] of Object.entries(searchParams)) {
      if (!Array.isArray(value) || !["keyIds", "names", "identities"].includes(field)) {
        continue;
      }

      for (const filterItem of value) {
        const baseFilter = {
          id: crypto.randomUUID(),
          field: field as KeysListFilterField,
          operator: filterItem.operator,
          value: filterItem.value,
        };

        activeFilters.push(baseFilter);
      }
    }

    return activeFilters;
  }, [searchParams]);

  const updateFilters = useCallback(
    (newFilters: KeysListFilterValue[]) => {
      const newParams: Partial<KeysQuerySearchParams> = {
        keyIds: null,
        names: null,
        identities: null,
      };

      const keyIdFilters: AllOperatorsUrlValue[] = [];
      const nameFilters: AllOperatorsUrlValue[] = [];
      const identitiesFilters: AllOperatorsUrlValue[] = [];

      newFilters.forEach((filter) => {
        const fieldConfig = keysListFilterFieldConfig[filter.field];
        const validOperators = fieldConfig.operators;
        const operator = validOperators.includes(filter.operator)
          ? filter.operator
          : validOperators[0];

        switch (filter.field) {
          case "keyIds":
            if (typeof filter.value === "string") {
              keyIdFilters.push({
                value: filter.value,
                operator: operator as "is" | "contains" | "startsWith" | "endsWith",
              });
            }
            break;
          case "names":
            if (typeof filter.value === "string") {
              nameFilters.push({
                value: filter.value,
                operator: operator as "is" | "contains" | "startsWith" | "endsWith",
              });
            }
            break;
          case "identities":
            if (typeof filter.value === "string") {
              identitiesFilters.push({
                value: filter.value,
                operator: operator as "is" | "contains" | "startsWith" | "endsWith",
              });
            }
            break;
        }
      });

      newParams.keyIds = keyIdFilters.length > 0 ? keyIdFilters : null;
      newParams.names = nameFilters.length > 0 ? nameFilters : null;
      newParams.identities = identitiesFilters.length > 0 ? identitiesFilters : null;

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
