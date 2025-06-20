import { z } from "zod";
import type { FieldConfig } from "../filter.types";
import { isNumberConfig, isStringConfig } from "./type-guards";

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
            const config = filterFieldConfig[data.field as keyof TConfig];
            return data.filters.every((filter) => {
              const isOperatorValid = config.operators.includes(
                filter.operator as z.infer<TOperatorEnum>,
              );
              return (
                isOperatorValid &&
                validateFieldValue(data.field as keyof TConfig, filter.value, filterFieldConfig)
              );
            });
          },
          { message: "Invalid field/operator/value combination" },
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

  return true;
}
