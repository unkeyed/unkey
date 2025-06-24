import type { FilterValue } from "@/components/logs/validation/filter.types";
import {
  parseAsFilterValueArray,
  parseAsRelativeTime,
} from "@/components/logs/validation/utils/nuqs-parsers";
import { createFilterOutputSchema } from "@/components/logs/validation/utils/structured-output-schema-generator";
import { parseAsInteger } from "nuqs";
import { z } from "zod";

// ============================================================================
// COMMON OPERATOR DEFINITIONS
// ============================================================================

export const COMMON_STRING_OPERATORS = ["is", "contains", "startsWith", "endsWith"] as const;

/**
 * Base configuration for any field type
 */
export interface BaseFieldConfig<TOperators extends readonly string[]> {
  type: "string" | "number";
  operators: TOperators;
}

interface TimeField {
  isTimeField: true;
}

/**
 * Special relative time field marker - uses parseAsRelativeTime parser
 */
interface RelativeTimeField {
  isRelativeTimeField: true;
}

/**
 * Complete field configuration type
 */
type FieldConfig<TOperators extends readonly string[]> =
  | BaseFieldConfig<TOperators>
  | (BaseFieldConfig<TOperators> & TimeField)
  | (BaseFieldConfig<TOperators> & RelativeTimeField);

// ============================================================================
// EXTRACTED TYPE HELPERS
// ============================================================================

/**
 * Extract all operators from field configurations
 */
type ExtractAllOperators<TConfigs extends Record<string, FieldConfig<readonly string[]>>> = {
  [K in keyof TConfigs]: TConfigs[K]["operators"][number];
}[keyof TConfigs];

/**
 * Check if field is a special time field
 */
type IsTimeField<T> = T extends { isTimeField: true } ? true : false;

/**
 * Check if field is a relative time field
 */
type IsRelativeTimeField<T> = T extends { isRelativeTimeField: true } ? true : false;

/**
 * Get operators for a specific field
 */
type GetFieldOperators<
  TConfigs extends Record<string, FieldConfig<readonly string[]>>,
  TField extends keyof TConfigs,
> = TConfigs[TField]["operators"][number];

/**
 * URL filter value for a specific field
 */
type FilterUrlValue<
  TConfigs extends Record<string, FieldConfig<readonly string[]>>,
  TField extends keyof TConfigs,
> = {
  operator: GetFieldOperators<TConfigs, TField>;
  value: TConfigs[TField]["type"] extends "number" ? number : string;
};

/**
 * URL value type for a field (handles special fields)
 */
type FieldUrlValueType<
  TConfigs extends Record<string, FieldConfig<readonly string[]>>,
  TField extends keyof TConfigs,
> = IsTimeField<TConfigs[TField]> extends true
  ? TConfigs[TField]["type"] extends "number"
    ? number | null
    : string | null
  : IsRelativeTimeField<TConfigs[TField]> extends true
    ? string | null
    : FilterUrlValue<TConfigs, TField>[] | null;

/**
 * Query search params type
 */
type QuerySearchParamsType<TConfigs extends Record<string, FieldConfig<readonly string[]>>> = {
  [K in keyof TConfigs]: FieldUrlValueType<TConfigs, K>;
};

/**
 * Base schema shape for field configurations
 */
type BaseSchemaShape<TConfigs extends Record<string, FieldConfig<readonly string[]>>> = {
  [K in keyof TConfigs]: IsTimeField<TConfigs[K]> extends true
    ? TConfigs[K]["type"] extends "number"
      ? z.ZodNumber
      : z.ZodString
    : IsRelativeTimeField<TConfigs[K]> extends true
      ? z.ZodString
      : z.ZodNullable<
          z.ZodObject<{
            filters: z.ZodArray<
              z.ZodObject<{
                operator: TConfigs[K]["operators"]["length"] extends 1
                  ? z.ZodLiteral<TConfigs[K]["operators"][0]>
                  : //@ts-expect-error safe to ignore
                    z.ZodEnum<TConfigs[K]["operators"]>;
                value: TConfigs[K]["type"] extends "number" ? z.ZodNumber : z.ZodString;
              }>
            >;
          }>
        >;
};

/**
 * Pagination schema shape
 */
type PaginationSchemaShape = {
  limit: z.ZodNumber;
  cursor: z.ZodOptional<z.ZodNullable<z.ZodNumber>>;
};

/**
 * Compile-time API Schema Shape type - conditionally includes pagination
 */
type BuildApiSchemaShape<
  TConfigs extends Record<string, FieldConfig<readonly string[]>>,
  TPagination extends boolean = false,
> = TPagination extends true
  ? PaginationSchemaShape & BaseSchemaShape<TConfigs>
  : BaseSchemaShape<TConfigs>;

/**
 * Options for createFilterSchema
 */
type CreateFilterSchemaOptions = {
  pagination?: boolean;
};

/**
 * Creates a complete filter schema from field configurations
 * This is the factory function that generates everything
 */
export function createFilterSchema<
  TPrefix extends string,
  TConfigs extends Record<string, FieldConfig<readonly string[]>>,
  TOptions extends CreateFilterSchemaOptions = CreateFilterSchemaOptions,
>(prefix: TPrefix, fieldConfigs: TConfigs, options: TOptions = {} as TOptions) {
  type AllOperators = ExtractAllOperators<TConfigs>;
  type FilterValueForField<TField extends keyof TConfigs> = FilterUrlValue<TConfigs, TField>;

  const fieldNames = Object.keys(fieldConfigs) as (keyof TConfigs)[];

  if (fieldNames.length === 0) {
    throw new Error(`${prefix}FilterFieldConfig must contain at least one field definition.`);
  }

  // Extract all unique operators
  const allOperators = Array.from(
    new Set(Object.values(fieldConfigs).flatMap((config) => config.operators)),
  ) as AllOperators[];

  const operatorEnum = z.enum(allOperators as [AllOperators, ...AllOperators[]]);

  const [firstFieldName, ...restFieldNames] = fieldNames;

  //@ts-expect-error safe to ignore
  const fieldEnum = z.enum([firstFieldName, ...restFieldNames] as [
    keyof TConfigs,
    ...(keyof TConfigs)[],
  ]);

  const queryParamsPayload = Object.fromEntries(
    fieldNames.map((fieldName) => {
      const config = fieldConfigs[fieldName];

      if ("isTimeField" in config && config.isTimeField) {
        return [fieldName, parseAsInteger];
      }

      if ("isRelativeTimeField" in config && config.isRelativeTimeField) {
        return [fieldName, parseAsRelativeTime];
      }

      // Regular fields use parseAsFilterValueArray
      //@ts-expect-error safe to ignore
      return [fieldName, parseAsFilterValueArray(config.operators)];
    }),
  );

  function createApiQuerySchema() {
    const schemaDefinition = {} as Record<string, z.ZodTypeAny>;

    // Add pagination fields only if requested
    if (options.pagination) {
      schemaDefinition.limit = z.number().int();
      schemaDefinition.cursor = z.number().nullable().optional();
    }

    // Process each field with exact type matching
    fieldNames.forEach((fieldName) => {
      const config = fieldConfigs[fieldName];

      if ("isTimeField" in config && config.isTimeField) {
        const fieldSchema = config.type === "number" ? z.number().int() : z.string();
        schemaDefinition[fieldName as string] = fieldSchema;
        return;
      }

      if ("isRelativeTimeField" in config && config.isRelativeTimeField) {
        schemaDefinition[fieldName as string] = z.string();
        return;
      }

      // Regular filter fields
      const operatorSchema =
        config.operators.length === 1
          ? z.literal(config.operators[0])
          : z.enum(config.operators as [string, ...string[]]);

      const valueSchema = config.type === "number" ? z.number() : z.string();

      schemaDefinition[fieldName as string] = z
        .object({
          filters: z.array(
            z.object({
              operator: operatorSchema,
              value: valueSchema,
            }),
          ),
        })
        .nullable();
    });

    return z.object(schemaDefinition);
  }

  const filterOutputSchema = createFilterOutputSchema(fieldEnum, operatorEnum, fieldConfigs);

  const apiQuerySchema = createApiQuerySchema();

  //@ts-expect-error safe to ignore
  const parseAsAllOperatorsFilterArray = parseAsFilterValueArray(allOperators);

  type OperatorType = z.infer<typeof operatorEnum>;
  type FieldType = z.infer<typeof fieldEnum>;

  //@ts-expect-error safe to ignore
  type FilterValueType = FilterValue<FieldType, OperatorType>;
  type AllOperatorsUrlValueType = FilterValueForField<keyof TConfigs>;

  return {
    // Configuration
    fieldConfigs,
    fieldNames,

    // Enums
    operatorEnum,
    fieldEnum,

    // Schemas
    filterOutputSchema,
    apiQuerySchema: apiQuerySchema as TOptions["pagination"] extends true
      ? z.ZodObject<BuildApiSchemaShape<TConfigs, true>>
      : z.ZodObject<BuildApiSchemaShape<TConfigs, false>>,

    // Parsers
    queryParamsPayload,
    parseAsAllOperatorsFilterArray,

    types: {} as {
      Operator: OperatorType;
      Field: FieldType;
      FilterValue: FilterValueType;
      QuerySearchParams: QuerySearchParamsType<TConfigs>;
      AllOperatorsUrlValue: AllOperatorsUrlValueType;
    },
  };
}
