import { z } from "zod";
import type {
  FieldConfig,
  FilterField,
  FilterFieldConfigs,
  FilterValue,
  NumberConfig,
  StringConfig,
} from "./filters.type";

export const filterOperatorEnum = z.enum(["is", "contains"]);

export const filterFieldEnum = z.enum([
  "startTime",
  "endTime",
  "since",
  "identifiers",
  "requestIds",
  "status",
]);

export const filterOutputSchema = z.object({
  filters: z.array(
    z
      .object({
        field: filterFieldEnum,
        filters: z.array(
          z.object({
            operator: filterOperatorEnum,
            value: z.union([z.string(), z.number()]),
          }),
        ),
      })
      .refine(
        (data) => {
          const config = filterFieldConfig[data.field];
          return data.filters.every((filter) => {
            const isOperatorValid = config.operators.includes(filter.operator as any);
            if (!isOperatorValid) {
              return false;
            }
            return validateFieldValue(data.field, filter.value);
          });
        },
        {
          message: "Invalid field/operator/value combination",
        },
      ),
  ),
});

// Required for transforming OpenAI structured outputs into our own Filter types
export const transformStructuredOutputToFilters = (
  data: z.infer<typeof filterOutputSchema>,
  existingFilters: FilterValue[] = [],
): FilterValue[] => {
  const uniqueFilters = [...existingFilters];
  const seenFilters = new Set(existingFilters.map((f) => `${f.field}-${f.operator}-${f.value}`));

  for (const filterGroup of data.filters) {
    filterGroup.filters.forEach((filter) => {
      const baseFilter = {
        field: filterGroup.field,
        operator: filter.operator,
        value: filter.value,
      };

      const filterKey = `${baseFilter.field}-${baseFilter.operator}-${baseFilter.value}`;

      if (seenFilters.has(filterKey)) {
        return;
      }

      uniqueFilters.push({
        id: crypto.randomUUID(),
        ...baseFilter,
      });

      seenFilters.add(filterKey);
    });
  }

  return uniqueFilters;
};

// Type guard for config types
function isNumberConfig(config: FieldConfig): config is NumberConfig {
  return config.type === "number";
}

function isStringConfig(config: FieldConfig): config is StringConfig {
  return config.type === "string";
}

export function validateFieldValue(field: FilterField, value: string | number): boolean {
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

export const filterFieldConfig: FilterFieldConfigs = {
  startTime: {
    type: "number",
    operators: ["is"],
  },
  endTime: {
    type: "number",
    operators: ["is"],
  },
  since: {
    type: "string",
    operators: ["is"],
  },
  identifiers: {
    type: "string",
    operators: ["is", "contains"],
  },
  requestIds: {
    type: "string",
    operators: ["is"],
  },
  status: {
    type: "string",
    operators: ["is"],
    validValues: ["rejected", "succeeded"],
  },
} as const;
