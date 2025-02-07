import type {
  FilterValue,
  NumberConfig,
  StringConfig,
} from "@/components/logs/validation/filter.types";
import { createFilterOutputSchema } from "@/components/logs/validation/utils/structured-output-schema-generator";
import { z } from "zod";

// Configuration
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
    validValues: ["blocked", "passed"],
    getColorClass: (value) => (value === "blocked" ? "bg-warning-9" : "bg-success-9"),
  } as const,
};

// Schemas
export const filterOperatorEnum = z.enum(["is", "contains"]);
export const filterFieldEnum = z.enum([
  "startTime",
  "endTime",
  "since",
  "identifiers",
  "requestIds",
  "status",
]);
export const filterOutputSchema = createFilterOutputSchema(
  filterFieldEnum,
  filterOperatorEnum,
  filterFieldConfig,
);

// Types
export type FilterOperator = z.infer<typeof filterOperatorEnum>;
export type FilterField = z.infer<typeof filterFieldEnum>;
export type FilterOutputSchema = z.infer<typeof filterOutputSchema>;

export type FilterFieldConfigs = {
  startTime: NumberConfig<FilterOperator>;
  endTime: NumberConfig<FilterOperator>;
  since: StringConfig<FilterOperator>;
  identifiers: StringConfig<FilterOperator>;
  requestIds: StringConfig<FilterOperator>;
  status: StringConfig<FilterOperator>;
};

export type FilterUrlValue = Pick<FilterValue<FilterField, FilterOperator>, "value" | "operator">;
export type RatelimitFilterValue = FilterValue<FilterField, FilterOperator>;

export type QuerySearchParams = {
  startTime?: number | null;
  endTime?: number | null;
  since?: string | null;
  identifiers: FilterUrlValue[] | null;
  requestIds: FilterUrlValue[] | null;
  status: FilterUrlValue[] | null;
};
