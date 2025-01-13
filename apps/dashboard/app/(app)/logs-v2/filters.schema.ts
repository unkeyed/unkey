import { z } from "zod";
import {
  FieldConfig,
  FilterField,
  FilterFieldConfigs,
  HttpMethod,
  NumberConfig,
  StatusConfig,
  StringConfig,
} from "./filters.type";
import { METHODS } from "./constants";

export const filterOperatorEnum = z.enum([
  "is",
  "contains",
  "startsWith",
  "endsWith",
]);

export const filterFieldEnum = z.enum([
  "host",
  "requestId",
  "methods",
  "paths",
  "status",
  "startTime",
  "endTime",
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
          })
        ),
      })
      .refine(
        (data) => {
          const config = filterFieldConfig[data.field];
          return data.filters.every((filter) => {
            const isOperatorValid = config.operators.includes(
              filter.operator as any
            );
            if (!isOperatorValid) return false;
            return validateFieldValue(data.field, filter.value);
          });
        },
        {
          message: "Invalid field/operator/value combination",
        }
      )
  ),
});

// Type guard for config types
function isStatusConfig(config: FieldConfig): config is StatusConfig {
  return "validate" in config && config.type === "number";
}

function isNumberConfig(config: FieldConfig): config is NumberConfig {
  return config.type === "number";
}

function isStringConfig(config: FieldConfig): config is StringConfig {
  return config.type === "string";
}

function validateFieldValue(
  field: FilterField,
  value: string | number
): boolean {
  const config = filterFieldConfig[field];

  if (isStatusConfig(config) && typeof value === "number") {
    return config.validate(value);
  }

  if (field === "methods" && typeof value === "string") {
    return METHODS.includes(value as HttpMethod);
  }

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
  status: {
    type: "number",
    operators: ["is"],
    getColorClass: (value) => {
      if (value >= 500) return "bg-error-9";
      if (value >= 400) return "bg-warning-8";
      return "bg-success-9";
    },
    validate: (value) => value >= 100 && value <= 599,
  },
  methods: {
    type: "string",
    operators: ["is"],
    validValues: METHODS,
  },
  paths: {
    type: "string",
    operators: ["is", "contains", "startsWith", "endsWith"],
  },
  host: {
    type: "string",
    operators: ["is", "contains"],
  },
  requestId: {
    type: "string",
    operators: ["is"],
  },
  startTime: {
    type: "number",
    operators: ["is"],
  },
  endTime: {
    type: "number",
    operators: ["is"],
  },
} as const;
