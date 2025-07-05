import { type UseQueryStatesKeysMap, useQueryStates } from "nuqs";
import { useCallback, useMemo } from "react";

type FilterConfig = {
  type: "string" | "number";
  operators: readonly string[];
  isTimeField?: boolean;
  isRelativeTimeField?: boolean;
  getColorClass?: (value: unknown) => string;
};

type FilterConfigs = Record<string, FilterConfig>;

type FilterValue<TField extends string, TOperator extends string> = {
  id: string;
  field: TField;
  operator: TOperator;
  value: string | number;
  metadata?: {
    colorClass?: string;
  };
};

type UrlFilterValue = {
  operator: string;
  value: string | number;
};

export function createUseFilters<
  TFilterSchema extends {
    fieldConfigs: FilterConfigs;
    queryParamsPayload: UseQueryStatesKeysMap;
    types: {
      Operator: string;
      Field: string;
      FilterValue: FilterValue<string, string>;
      QuerySearchParams: Record<string, unknown>;
    };
  },
>(filterSchema: TFilterSchema) {
  return function useFilters() {
    const [searchParams, setSearchParams] = useQueryStates(filterSchema.queryParamsPayload, {
      history: "push",
    });

    const filters = useMemo(() => {
      const activeFilters: TFilterSchema["types"]["FilterValue"][] = [];

      Object.entries(searchParams).forEach(([fieldName, fieldValue]) => {
        const field = fieldName as TFilterSchema["types"]["Field"];
        const config = filterSchema.fieldConfigs[field];

        if (!config || fieldValue === null || fieldValue === undefined) {
          return;
        }

        // Handle time fields (direct values)
        if (config.isTimeField) {
          activeFilters.push({
            id: crypto.randomUUID(),
            field,
            operator: "is" as TFilterSchema["types"]["Operator"],
            value: fieldValue as number,
          } as TFilterSchema["types"]["FilterValue"]);
          return;
        }

        // Handle relative time fields (direct values)
        if (config.isRelativeTimeField) {
          activeFilters.push({
            id: crypto.randomUUID(),
            field,
            operator: "is" as TFilterSchema["types"]["Operator"],
            value: fieldValue as string,
          } as TFilterSchema["types"]["FilterValue"]);
          return;
        }

        // Handle regular filter arrays
        if (Array.isArray(fieldValue)) {
          fieldValue.forEach((filterItem: UrlFilterValue) => {
            activeFilters.push({
              id: crypto.randomUUID(),
              field,
              operator: filterItem.operator as TFilterSchema["types"]["Operator"],
              value: filterItem.value,
              metadata: config.getColorClass
                ? {
                    colorClass: config.getColorClass(filterItem.value),
                  }
                : undefined,
            } as TFilterSchema["types"]["FilterValue"]);
          });
        }
      });

      return activeFilters;
    }, [searchParams, filterSchema]);

    const updateFilters = useCallback(
      (newFilters: TFilterSchema["types"]["FilterValue"][]) => {
        const newParams = {} as Record<TFilterSchema["types"]["Field"], unknown>;

        // Initialize all fields to null
        Object.keys(filterSchema.fieldConfigs).forEach((field) => {
          newParams[field as TFilterSchema["types"]["Field"]] = null;
        });

        // Group filters by field
        const filterGroups: Record<
          TFilterSchema["types"]["Field"],
          TFilterSchema["types"]["FilterValue"][]
        > = {} as Record<TFilterSchema["types"]["Field"], TFilterSchema["types"]["FilterValue"][]>;

        newFilters.forEach((filter) => {
          const field = filter.field;
          //@ts-expect-error safe to ignore
          if (!filterGroups[field]) {
            //@ts-expect-error safe to ignore
            filterGroups[field] = [];
          }
          //@ts-expect-error safe to ignore
          filterGroups[field].push(filter);
        });

        // Convert filter groups to URL params
        (
          Object.entries(filterGroups) as [
            TFilterSchema["types"]["Field"],
            TFilterSchema["types"]["FilterValue"][],
          ][]
        ).forEach(([field, filters]) => {
          const config = filterSchema.fieldConfigs[field];

          if (config.isTimeField) {
            const timeFilter = filters[0];
            if (timeFilter) {
              newParams[field] = timeFilter.value as number;
            }
            return;
          }

          if (config.isRelativeTimeField) {
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

        setSearchParams(newParams as Partial<TFilterSchema["types"]["QuerySearchParams"]>);
      },
      [setSearchParams, filterSchema],
    );

    const removeFilter = useCallback(
      (id: string) => {
        const newFilters = filters.filter((f) => f.id !== id);
        updateFilters(newFilters);
      },
      [filters, updateFilters],
    );

    const addFilter = useCallback(
      (filter: Omit<TFilterSchema["types"]["FilterValue"], "id">) => {
        const newFilter: TFilterSchema["types"]["FilterValue"] = {
          ...filter,
          id: crypto.randomUUID(),
        } as TFilterSchema["types"]["FilterValue"];
        updateFilters([...filters, newFilter]);
      },
      [filters, updateFilters],
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
}
