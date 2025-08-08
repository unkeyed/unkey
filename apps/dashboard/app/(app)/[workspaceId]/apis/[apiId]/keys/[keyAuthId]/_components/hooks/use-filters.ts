// src/features/keys/hooks/use-filters.ts (or your path)

import { useQueryStates } from "nuqs";
import { useCallback, useMemo } from "react";
import {
  type AllOperatorsUrlValue,
  type KeysListFilterField,
  type KeysListFilterValue,
  type KeysQuerySearchParams,
  keysListFilterFieldConfig,
  keysListFilterFieldNames,
  parseAsAllOperatorsFilterArray,
} from "../filters.schema";

export const queryParamsPayload = Object.fromEntries(
  keysListFilterFieldNames.map((field) => [field, parseAsAllOperatorsFilterArray]),
) as { [K in KeysListFilterField]: typeof parseAsAllOperatorsFilterArray };

export const useFilters = () => {
  const [searchParams, setSearchParams] = useQueryStates(queryParamsPayload, {
    history: "push",
  });

  const filters = useMemo(() => {
    const activeFilters: KeysListFilterValue[] = [];

    for (const field of keysListFilterFieldNames) {
      const value = searchParams[field];

      if (!Array.isArray(value)) {
        continue;
      }

      for (const filterItem of value) {
        if (filterItem && typeof filterItem.value === "string" && filterItem.operator) {
          const baseFilter: KeysListFilterValue = {
            id: crypto.randomUUID(),
            field: field,
            operator: filterItem.operator,
            value: filterItem.value,
          };
          activeFilters.push(baseFilter);
        }
      }
    }

    return activeFilters;
  }, [searchParams]);

  const updateFilters = useCallback(
    (newFilters: KeysListFilterValue[]) => {
      const newParams: Partial<KeysQuerySearchParams> = Object.fromEntries(
        keysListFilterFieldNames.map((field) => [field, null]),
      );

      const filtersByField = new Map<KeysListFilterField, AllOperatorsUrlValue[]>();
      keysListFilterFieldNames.forEach((field) => filtersByField.set(field, []));

      newFilters.forEach((filter) => {
        if (!keysListFilterFieldNames.includes(filter.field)) {
          return;
        }

        const fieldConfig = keysListFilterFieldConfig[filter.field];
        const validOperators = fieldConfig.operators;

        if (!validOperators.includes(filter.operator)) {
          throw new Error("Invalid operator");
        }

        if (typeof filter.value === "string") {
          const fieldFilters = filtersByField.get(filter.field);
          fieldFilters?.push({
            value: filter.value,
            operator: filter.operator,
          });
        }
      });

      filtersByField.forEach((fieldFilters, field) => {
        if (fieldFilters.length > 0) {
          newParams[field] = fieldFilters;
        }
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
