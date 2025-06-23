import type { FilterValue } from "@/components/logs/validation/filter.types";
import { parseAsFilterValueArray } from "@/components/logs/validation/utils/nuqs-parsers";
import { createFilterOutputSchema } from "@/components/logs/validation/utils/structured-output-schema-generator";
import { parseAsInteger, useQueryStates } from "nuqs";
import { useMemo, useCallback } from "react";
import { z } from "zod";

export const COMMON_STRING_OPERATORS = [
  "is",
  "contains",
  "startsWith",
  "endsWith",
] as const;

export interface BaseFieldConfig<T extends readonly string[]> {
  type: "string" | "number";
  operators: T;
  validValues?: readonly string[];
  getColorClass?: (value: unknown) => string;
  validate?: (value: unknown) => boolean;
}

export function createFilterSchema<
  TOperators extends readonly [string, ...string[]],
  TFields extends readonly [string, ...string[]]
>(
  operators: TOperators,
  fields: TFields,
  fieldConfigs: Record<
    TFields[number],
    BaseFieldConfig<readonly TOperators[number][]>
  >
) {
  const operatorEnum = z.enum(operators);
  const fieldEnum = z.enum(fields);

  const filterOutputSchema = createFilterOutputSchema(
    // @ts-expect-error Safe to ignore
    fieldEnum,
    operatorEnum,
    fieldConfigs
  );

  type OperatorType = z.infer<typeof operatorEnum>;
  type FieldType = z.infer<typeof fieldEnum>;
  // @ts-expect-error Safe to ignore
  type FilterValueType = FilterValue<FieldType, OperatorType>;

  type UrlValueType = {
    value: string | number;
    operator: OperatorType;
  };

  type QuerySearchParamsType = {
    [K in TFields[number]]: (typeof fieldConfigs)[K] extends {
      type: "number";
      operators: readonly ["is"];
    }
      ? number | null
      : UrlValueType[] | null;
  } & {
    startTime?: number | null;
    endTime?: number | null;
    since?: string | null;
  };

  // Create parsers
  const createQueryParamsPayload = (): Record<string, unknown> => {
    const payload: Record<string, unknown> = {};

    for (const fieldName of fields) {
      const fieldConfig = fieldConfigs[fieldName as TFields[number]];

      if (
        fieldConfig.type === "number" &&
        fieldConfig.operators.length === 1 &&
        fieldConfig.operators[0] === "is"
      ) {
        payload[fieldName] = parseAsInteger;
      } else {
        // @ts-expect-error Zod is going mad here
        payload[fieldName] = parseAsFilterValueArray(fieldConfig.operators);
      }
    }

    return payload;
  };

  return {
    filterFieldConfig: fieldConfigs,
    fieldNames: fields,

    operatorEnum,
    fieldEnum,
    filterOutputSchema,

    // @ts-expect-error Safe to ignore
    parseAsFilterArray: parseAsFilterValueArray(operators),
    queryParamsPayload: createQueryParamsPayload(),

    types: {} as {
      Operator: OperatorType;
      Field: FieldType;
      FilterValue: FilterValueType;
      UrlValue: UrlValueType;
      QuerySearchParams: QuerySearchParamsType;
    },
  };
}

interface FilterSchema {
  filterFieldConfig: Record<string, BaseFieldConfig<readonly string[]>>;
  fieldNames: readonly string[];
  parseAsFilterArray: ReturnType<typeof parseAsFilterValueArray>;
  // biome-ignore lint/suspicious/noExplicitAny: <explanation>
  queryParamsPayload: Record<string, any>;
  types: {
    Field: string;
    Operator: string;
    FilterValue: {
      id: string;
      field: string;
      operator: string;
      value: string | number;
    };
    UrlValue: {
      value: string | number;
      operator: string;
    };
    // biome-ignore lint/suspicious/noExplicitAny: <explanation>
    QuerySearchParams: any;
  };
}

// Generic hook factory with proper type constraints
export function createUseFilters<T extends FilterSchema>(schema: T) {
  return function useFilters() {
    const [searchParams, setSearchParams] = useQueryStates(
      schema.queryParamsPayload,
      {
        history: "push",
      }
    );

    const filters = useMemo(() => {
      const activeFilters: T["types"]["FilterValue"][] = [];

      for (const field of schema.fieldNames) {
        const value = searchParams[field];
        if (!Array.isArray(value)) {
          continue;
        }

        for (const filterItem of value) {
          if (
            filterItem &&
            typeof filterItem === "object" &&
            "value" in filterItem &&
            "operator" in filterItem &&
            (typeof filterItem.value === "string" ||
              typeof filterItem.value === "number") &&
            typeof filterItem.operator === "string"
          ) {
            activeFilters.push({
              id: crypto.randomUUID(),
              field: field as T["types"]["Field"],
              operator: filterItem.operator as T["types"]["Operator"],
              value: filterItem.value,
            } as T["types"]["FilterValue"]);
          }
        }
      }
      return activeFilters;
    }, [searchParams]);

    const updateFilters = useCallback(
      (newFilters: T["types"]["FilterValue"][]) => {
        const newParams: Record<string, unknown> = Object.fromEntries(
          schema.fieldNames.map((field) => [field, null])
        );

        const filtersByField = new Map<string, T["types"]["UrlValue"][]>();
        schema.fieldNames.forEach((field) => filtersByField.set(field, []));

        for (const filter of newFilters) {
          if (!schema.fieldNames.includes(filter.field)) {
            throw new Error(`Invalid filter field: ${filter.field}`);
          }

          const fieldConfig = schema.filterFieldConfig[filter.field];
          if (!fieldConfig) {
            throw new Error(
              `No configuration found for field: ${filter.field}`
            );
          }

          if (!fieldConfig.operators.includes(filter.operator)) {
            throw new Error(
              `Invalid operator '${filter.operator}' for field '${
                filter.field
              }'. Valid operators: ${fieldConfig.operators.join(", ")}`
            );
          }

          // biome-ignore lint/suspicious/useValidTypeof: <explanation>
          if (typeof filter.value !== fieldConfig.type) {
            throw new Error(
              `Filter value must be a ${fieldConfig.type} for field '${
                filter.field
              }', got ${typeof filter.value}`
            );
          }

          const fieldFilters = filtersByField.get(filter.field);
          if (!fieldFilters) {
            throw new Error(
              `Failed to get filters for field '${filter.field}'`
            );
          }

          fieldFilters.push({
            value: filter.value,
            operator: filter.operator as T["types"]["Operator"],
          });
        }

        // Set non-empty filter arrays in params
        filtersByField.forEach((fieldFilters, field) => {
          if (fieldFilters.length > 0) {
            newParams[field] = fieldFilters;
          }
        });

        setSearchParams(newParams);
      },
      [setSearchParams, schema]
    );

    const removeFilter = useCallback(
      (id: string) => {
        const newFilters = filters.filter((f) => f.id !== id);
        updateFilters(newFilters);
      },
      [filters, updateFilters]
    );

    return {
      filters,
      removeFilter,
      updateFilters,
    };
  };
}
