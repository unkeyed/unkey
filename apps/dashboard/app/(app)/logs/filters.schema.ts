import { z } from "zod";
import { METHODS } from "./constants";
import type {
  FieldConfig,
  FilterField,
  FilterFieldConfigs,
  FilterValue,
  HttpMethod,
  NumberConfig,
  StatusConfig,
  StringConfig,
} from "./filters.type";

export const filterOperatorEnum = z.enum(["is", "contains", "startsWith", "endsWith"]);

export const filterFieldEnum = z.enum([
  "host",
  "requestId",
  "methods",
  "paths",
  "status",
  "startTime",
  "endTime",
  "since",
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

      if (filterGroup.field === "status") {
        const numericValue =
          typeof filter.value === "string" ? Number.parseInt(filter.value) : filter.value;

        uniqueFilters.push({
          id: crypto.randomUUID(),
          ...baseFilter,
          value: numericValue,
          metadata: {
            colorClass: filterFieldConfig.status.getColorClass?.(numericValue),
          },
        });
      } else {
        uniqueFilters.push({
          id: crypto.randomUUID(),
          ...baseFilter,
        });
      }

      seenFilters.add(filterKey);
    });
  }

  return uniqueFilters;
};

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

export function validateFieldValue(field: FilterField, value: string | number): boolean {
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
      if (value >= 500) {
        return "bg-error-9";
      }
      if (value >= 400) {
        return "bg-warning-8";
      }
      return "bg-success-9";
    },
    validate: (value) => value >= 200 && value <= 599,
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
    operators: ["is"],
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
  since: {
    type: "string",
    operators: ["is"],
  },
} as const;
