import { z } from "zod";
import type { FieldConfig } from "../filter.types";
import { isNumberConfig, isStringConfig } from "./type-guards";

interface FilterItem {
  operator: string;
  value: string | number;
}

interface FilterData {
  field: string;
  filters: FilterItem[];
}

export function createFilterOutputSchema<
  TFieldEnum extends z.ZodEnum<Readonly<Record<string, string>>>,
  TOperatorEnum extends z.ZodEnum<Readonly<Record<string, string>>>,
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
        .superRefine((data, ctx) => {
          // Type assertion is safe here because the schema structure guarantees these properties exist
          const typedData = data as unknown as FilterData;
          const config = filterFieldConfig[typedData.field as keyof TConfig];
          typedData.filters.forEach((filter, index: number) => {
            const isOperatorValid = config.operators.includes(
              filter.operator as z.infer<TOperatorEnum>,
            );
            if (!isOperatorValid) {
              ctx.addIssue({
                code: "custom",
                message: `Invalid operator "${filter.operator}" for field "${String(typedData.field)}"`,
                path: ["filters", index, "operator"],
              });
            }

            const isValueValid = validateFieldValue(
              typedData.field as keyof TConfig,
              filter.value,
              filterFieldConfig,
            );
            if (!isValueValid) {
              ctx.addIssue({
                code: "custom",
                message: `Invalid value "${filter.value}" for field "${String(typedData.field)}"`,
                path: ["filters", index, "value"],
              });
            }
          });
        }),
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
