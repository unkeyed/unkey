import { z } from "zod";
import type { FieldConfig } from "../filter.types";
import { isNumberConfig, isStringConfig } from "./type-guards";

// Helper function to validate a single filter and return detailed result
function validateSingleFilter<
  TFieldEnum extends z.ZodEnum<[string, ...string[]]>,
  TOperatorEnum extends z.ZodEnum<[string, ...string[]]>,
  TConfig extends Record<z.infer<TFieldEnum>, FieldConfig<z.infer<TOperatorEnum>>>,
>(
  field: keyof TConfig,
  filter: { operator: z.infer<TOperatorEnum>; value: string | number },
  filterFieldConfig: TConfig,
) {
  const config = filterFieldConfig[field];

  const isOperatorValid = config.operators.includes(filter.operator);
  if (!isOperatorValid) {
    return {
      valid: false,
      error: `Invalid operator "${filter.operator}" for field "${String(field)}"`,
    };
  }

  const isValueValid = validateFieldValue(field, filter.value, filterFieldConfig);
  if (!isValueValid) {
    return {
      valid: false,
      error: `Invalid value "${filter.value}" for field "${String(field)}"`,
    };
  }

  return { valid: true };
}

export function createFilterOutputSchema<
  TFieldEnum extends z.ZodEnum<[string, ...string[]]>,
  TOperatorEnum extends z.ZodEnum<[string, ...string[]]>,
  TConfig extends Record<z.infer<TFieldEnum>, FieldConfig<z.infer<TOperatorEnum>>>,
>(fieldEnum: TFieldEnum, operatorEnum: TOperatorEnum, filterFieldConfig: TConfig) {
  return z.object({
    filters: z.array(
      z
        .object({
          field: fieldEnum,
          filters: z.array(
            z.object({
              operator: operatorEnum,
              value: z.union([z.string(), z.number()]),
            }),
          ),
        })
        .refine(
          (data) => {
            // Check if all filters are valid
            return data.filters.every(
              (filter) =>
                validateSingleFilter(
                  data.field as keyof TConfig,
                  filter as { operator: string; value: string | number },
                  filterFieldConfig,
                ).valid,
            );
          },
          (data) => {
            // Find the first invalid filter and return its error message
            const invalidFilter = data.filters.find(
              (filter) =>
                !validateSingleFilter(
                  data.field as keyof TConfig,
                  filter as { operator: string; value: string | number },
                  filterFieldConfig,
                ).valid,
            );

            if (invalidFilter) {
              const validation = validateSingleFilter(
                data.field as keyof TConfig,
                invalidFilter as { operator: string; value: string | number },
                filterFieldConfig,
              );
              return { message: validation.error || "Invalid field/operator/value combination" };
            }

            return { message: "Invalid field/operator/value combination" };
          },
        ),
    ),
  });
}

export function validateFieldValue<TConfig extends Record<string, FieldConfig>>(
  field: keyof TConfig,
  value: string | number,
  filterFieldConfig: TConfig,
): boolean {
  const config = filterFieldConfig[field];

  if (isStringConfig(config) && typeof value === "string") {
    if (config.validValues) {
      return config.validValues.includes(value);
    }
    return config.validate ? config.validate(value) : true;
  }

  if (isNumberConfig(config) && typeof value === "number") {
    return config.validate ? config.validate(value) : true;
  }

  return false;
}
