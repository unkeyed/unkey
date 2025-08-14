import { useQueryStates } from "nuqs";
import { useCallback, useMemo } from "react";
import {
  type DeploymentListFilterField,
  type DeploymentListFilterUrlValue,
  type DeploymentListFilterValue,
  deploymentListFilterFieldConfig,
  deploymentListFilterFieldNames,
  parseAsAllOperatorsFilterArray,
} from "../filters.schema";

// Only include fields that use filter arrays (exclude time fields)
const arrayFilterFields = deploymentListFilterFieldNames.filter(
  (field) => !["startTime", "endTime", "since"].includes(field),
) as Exclude<DeploymentListFilterField, "startTime" | "endTime" | "since">[];

export const queryParamsPayload = Object.fromEntries(
  arrayFilterFields.map((field) => [field, parseAsAllOperatorsFilterArray]),
) as {
  [K in Exclude<
    DeploymentListFilterField,
    "startTime" | "endTime" | "since"
  >]: typeof parseAsAllOperatorsFilterArray;
};

export const useFilters = () => {
  const [searchParams, setSearchParams] = useQueryStates(queryParamsPayload, {
    history: "push",
  });

  const filters = useMemo(() => {
    const activeFilters: DeploymentListFilterValue[] = [];

    for (const field of arrayFilterFields) {
      const value = searchParams[field];
      if (!Array.isArray(value)) {
        continue;
      }
      for (const filterItem of value) {
        if (filterItem && typeof filterItem.value === "string" && filterItem.operator) {
          const baseFilter: DeploymentListFilterValue = {
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
    (newFilters: DeploymentListFilterValue[]) => {
      const newParams: Record<string, DeploymentListFilterUrlValue[] | null> = Object.fromEntries(
        arrayFilterFields.map((field) => [field, null]),
      );

      const filtersByField = new Map<DeploymentListFilterField, DeploymentListFilterUrlValue[]>();
      deploymentListFilterFieldNames.forEach((field) => filtersByField.set(field, []));

      newFilters.forEach((filter) => {
        if (!deploymentListFilterFieldNames.includes(filter.field)) {
          throw new Error(`Invalid filter field: ${filter.field}`);
        }

        const fieldConfig = deploymentListFilterFieldConfig[filter.field];
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
