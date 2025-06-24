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

export const COMMON_STRING_OPERATORS = [
  "is",
  "contains",
  "startsWith",
  "endsWith",
] as const;
export const COMMON_NUMBER_OPERATORS = ["is"] as const;

// ============================================================================
// FIELD CONFIGURATION TYPES
// ============================================================================

/**
 * Base configuration for any field type
 */
export interface BaseFieldConfig<TOperators extends readonly string[]> {
  type: "string" | "number";
  operators: TOperators;
  validValues?: readonly string[];
  getColorClass?: (value: unknown) => string;
  validate?: (value: unknown) => boolean;
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
type ExtractAllOperators<
  TConfigs extends Record<string, FieldConfig<readonly string[]>>
> = {
  [K in keyof TConfigs]: TConfigs[K]["operators"][number];
}[keyof TConfigs];

/**
 * Check if field is a special time field
 */
type IsTimeField<T> = T extends { isTimeField: true } ? true : false;

/**
 * Check if field is a relative time field
 */
type IsRelativeTimeField<T> = T extends { isRelativeTimeField: true }
  ? true
  : false;

/**
 * Get operators for a specific field
 */
type GetFieldOperators<
  TConfigs extends Record<string, FieldConfig<readonly string[]>>,
  TField extends keyof TConfigs
> = TConfigs[TField]["operators"][number];

/**
 * URL filter value for a specific field
 */
type FilterUrlValue<
  TConfigs extends Record<string, FieldConfig<readonly string[]>>,
  TField extends keyof TConfigs
> = {
  operator: GetFieldOperators<TConfigs, TField>;
  value: TConfigs[TField]["type"] extends "number" ? number : string;
};

/**
 * URL value type for a field (handles special fields)
 */
type FieldUrlValueType<
  TConfigs extends Record<string, FieldConfig<readonly string[]>>,
  TField extends keyof TConfigs
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
type QuerySearchParamsType<
  TConfigs extends Record<string, FieldConfig<readonly string[]>>
> = {
  [K in keyof TConfigs]: FieldUrlValueType<TConfigs, K>;
};

/**
 * Compile-time API Schema Shape type - builds exact Zod types
 */
type BuildApiSchemaShape<
  TConfigs extends Record<string, FieldConfig<readonly string[]>>
> = {
  limit: z.ZodNumber;
  cursor: z.ZodOptional<z.ZodNullable<z.ZodNumber>>;
} & {
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
              value: TConfigs[K]["type"] extends "number"
                ? z.ZodNumber
                : z.ZodString;
            }>
          >;
        }>
      >;
};

/**
 * Creates a complete filter schema from field configurations
 * This is the main factory function that generates everything
 */
export function createFilterSchema<
  TPrefix extends string,
  TConfigs extends Record<string, FieldConfig<readonly string[]>>
>(prefix: TPrefix, fieldConfigs: TConfigs) {
  // ============================================================================
  // TYPE ALIASES FOR CLEANER CODE
  // ============================================================================

  type AllOperators = ExtractAllOperators<TConfigs>;
  type FilterValueForField<TField extends keyof TConfigs> = FilterUrlValue<
    TConfigs,
    TField
  >;

  // ============================================================================
  // VALIDATION AND SETUP
  // ============================================================================

  const fieldNames = Object.keys(fieldConfigs) as (keyof TConfigs)[];

  if (fieldNames.length === 0) {
    throw new Error(
      `${prefix}FilterFieldConfig must contain at least one field definition.`
    );
  }

  // Extract all unique operators
  const allOperators = Array.from(
    new Set(Object.values(fieldConfigs).flatMap((config) => config.operators))
  ) as AllOperators[];

  // ============================================================================
  // ZOD ENUMS
  // ============================================================================

  const operatorEnum = z.enum(
    allOperators as [AllOperators, ...AllOperators[]]
  );

  const [firstFieldName, ...restFieldNames] = fieldNames;

  //@ts-expect-error safe to ignore
  const fieldEnum = z.enum([firstFieldName, ...restFieldNames] as [
    keyof TConfigs,
    ...(keyof TConfigs)[]
  ]);

  // ============================================================================
  // QUERY PARAMS PAYLOAD GENERATION
  // ============================================================================

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
    })
  );

  // ============================================================================
  // API QUERY SCHEMA GENERATION WITH COMPILE-TIME TYPE PRESERVATION
  // ============================================================================

  function createApiQuerySchema(): z.ZodObject<BuildApiSchemaShape<TConfigs>> {
    // Build the schema shape at compile time
    type SchemaShape = BuildApiSchemaShape<TConfigs>;

    // Create the actual schema object that matches our compile-time type
    const schemaDefinition = {} as Record<string, z.ZodTypeAny>;

    // Base fields
    schemaDefinition.limit = z.number().int();
    schemaDefinition.cursor = z.number().nullable().optional();

    // Process each field with exact type matching
    fieldNames.forEach((fieldName) => {
      const config = fieldConfigs[fieldName];

      // FIXED: Check for isTimeField, not isSpecialTimeField
      if ("isTimeField" in config && config.isTimeField) {
        const fieldSchema =
          config.type === "number" ? z.number().int() : z.string();
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
            })
          ),
        })
        .nullable();
    });

    // Cast to our exact type - this preserves the compile-time type information
    return z.object(schemaDefinition as SchemaShape);
  }

  // ============================================================================
  // GENERATED SCHEMAS
  // ============================================================================

  const filterOutputSchema = createFilterOutputSchema(
    fieldEnum,
    operatorEnum,
    fieldConfigs
  );

  const apiQuerySchema = createApiQuerySchema();

  // ============================================================================
  // PARSER GENERATION
  // ============================================================================

  //@ts-expect-error safe to ignore
  const parseAsAllOperatorsFilterArray = parseAsFilterValueArray(allOperators);

  // ============================================================================
  // TYPE DEFINITIONS
  // ============================================================================

  type OperatorType = z.infer<typeof operatorEnum>;
  type FieldType = z.infer<typeof fieldEnum>;

  //@ts-expect-error safe to ignore
  type FilterValueType = FilterValue<FieldType, OperatorType>;
  type AllOperatorsUrlValueType = FilterValueForField<keyof TConfigs>;

  // ============================================================================
  // RETURN OBJECT
  // ============================================================================

  return {
    // Configuration
    fieldConfigs,
    fieldNames,

    // Enums
    operatorEnum,
    fieldEnum,

    // Schemas
    filterOutputSchema,
    apiQuerySchema,

    // Parsers
    queryParamsPayload,
    parseAsAllOperatorsFilterArray,

    // Type helpers (for export)
    types: {} as {
      Operator: OperatorType;
      Field: FieldType;
      FilterValue: FilterValueType;
      QuerySearchParams: QuerySearchParamsType<TConfigs>;
      AllOperatorsUrlValue: AllOperatorsUrlValueType;
    },
  };
}
