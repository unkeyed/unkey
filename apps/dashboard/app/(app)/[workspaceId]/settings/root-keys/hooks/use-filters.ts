import { useQueryStates } from "nuqs";
import { useCallback, useMemo } from "react";
import {
  type AllOperatorsUrlValue,
  type RootKeysFilterField,
  type RootKeysFilterValue,
  type RootKeysQuerySearchParams,
  parseAsAllOperatorsFilterArray,
  rootKeysFilterFieldConfig,
  rootKeysListFilterFieldNames,
} from "../filters.schema";

export const queryParamsPayload = Object.fromEntries(
  rootKeysListFilterFieldNames.map((field) => [field, parseAsAllOperatorsFilterArray]),
) as { [K in RootKeysFilterField]: typeof parseAsAllOperatorsFilterArray };

export const useFilters = () => {
  const [searchParams, setSearchParams] = useQueryStates(queryParamsPayload, {
    history: "push",
  });

  const filters = useMemo(() => {
    const activeFilters: RootKeysFilterValue[] = [];

    for (const field of rootKeysListFilterFieldNames) {
      const value = searchParams[field];
      if (!Array.isArray(value)) {
        continue;
      }

      value.forEach((filterItem, idx) => {
        const operator = filterItem?.operator;
        const raw = typeof filterItem?.value === "string" ? filterItem.value : "";
        const v = raw.trim();
        const allowed = rootKeysFilterFieldConfig[field]?.operators ?? [];
        if (!v || !operator || !allowed.includes(operator)) {
          return;
        }
        activeFilters.push({
          id: `${field}:${operator}:${v}:${idx}`, // deterministic across SSR/CSR
          field,
          operator,
          value: v,
        });
      });
    }

    return activeFilters;
  }, [searchParams]);

  const updateFilters = useCallback(
    (newFilters: RootKeysFilterValue[]) => {
      const newParams: Partial<RootKeysQuerySearchParams> = Object.fromEntries(
        rootKeysListFilterFieldNames.map((field) => [field, null]),
      );

      const filtersByField = new Map<RootKeysFilterField, AllOperatorsUrlValue[]>();
      rootKeysListFilterFieldNames.forEach((field) => filtersByField.set(field, []));

      newFilters.forEach((filter) => {
        if (!rootKeysListFilterFieldNames.includes(filter.field)) {
          throw new Error(`Invalid filter field: ${filter.field}`);
        }

        const fieldConfig = rootKeysFilterFieldConfig[filter.field];
        if (!fieldConfig.operators.includes(filter.operator)) {
          throw new Error(`Invalid operator '${filter.operator}' for field '${filter.field}'`);
        }

        if (typeof filter.value !== "string") {
          throw new Error(`Filter value must be a string for field '${filter.field}'`);
        }

        const fieldFilters = filtersByField.get(filter.field);
        if (!fieldFilters) {
          throw new Error(`Failed to get filters for field '${filter.field}'`);
        }

        fieldFilters.push({
          value: filter.value,
          operator: filter.operator,
        });
      });

      // Set non-empty filter arrays in params
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
