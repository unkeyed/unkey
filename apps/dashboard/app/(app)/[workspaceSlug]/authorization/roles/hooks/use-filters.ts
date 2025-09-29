import { useQueryStates } from "nuqs";
import { useCallback, useMemo } from "react";
import {
  type AllOperatorsUrlValue,
  type RolesFilterField,
  type RolesFilterValue,
  type RolesQuerySearchParams,
  parseAsAllOperatorsFilterArray,
  rolesFilterFieldConfig,
  rolesListFilterFieldNames,
} from "../filters.schema";

export const queryParamsPayload = Object.fromEntries(
  rolesListFilterFieldNames.map((field) => [field, parseAsAllOperatorsFilterArray]),
) as { [K in RolesFilterField]: typeof parseAsAllOperatorsFilterArray };

export const useFilters = () => {
  const [searchParams, setSearchParams] = useQueryStates(queryParamsPayload, {
    history: "push",
  });

  const filters = useMemo(() => {
    const activeFilters: RolesFilterValue[] = [];

    for (const field of rolesListFilterFieldNames) {
      const value = searchParams[field];
      if (!Array.isArray(value)) {
        continue;
      }

      for (const filterItem of value) {
        if (filterItem && typeof filterItem.value === "string" && filterItem.operator) {
          const baseFilter: RolesFilterValue = {
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
    (newFilters: RolesFilterValue[]) => {
      const newParams: Partial<RolesQuerySearchParams> = Object.fromEntries(
        rolesListFilterFieldNames.map((field) => [field, null]),
      );

      const filtersByField = new Map<RolesFilterField, AllOperatorsUrlValue[]>();
      rolesListFilterFieldNames.forEach((field) => filtersByField.set(field, []));

      newFilters.forEach((filter) => {
        if (!rolesListFilterFieldNames.includes(filter.field)) {
          throw new Error(`Invalid filter field: ${filter.field}`);
        }

        const fieldConfig = rolesFilterFieldConfig[filter.field];
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
