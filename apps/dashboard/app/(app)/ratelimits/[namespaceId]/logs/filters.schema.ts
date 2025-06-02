import type {
  FilterValue,
  NumberConfig,
  StringConfig,
} from "@/components/logs/validation/filter.types";
import { createFilterOutputSchema } from "@/components/logs/validation/utils/structured-output-schema-generator";
import { z } from "zod";

// Configuration
export const ratelimitFilterFieldConfig: FilterFieldConfigs = {
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
  status: {
    type: "string",
    operators: ["is"],
    validValues: ["blocked", "passed"],
    getColorClass: (value) => (value === "blocked" ? "bg-warning-9" : "bg-success-9"),
  } as const,
};

// Schemas
export const ratelimitFilterOperatorEnum = z.enum(["is", "contains"]);
export const ratelimitFilterFieldEnum = z.enum([
  "startTime",
  "endTime",
  "since",
  "identifiers",
  "status",
]);
export const filterOutputSchema = createFilterOutputSchema(
  ratelimitFilterFieldEnum,
  ratelimitFilterOperatorEnum,
  ratelimitFilterFieldConfig,
);

// Types
export type RatelimitFilterOperator = z.infer<typeof ratelimitFilterOperatorEnum>;
export type RatelimitFilterField = z.infer<typeof ratelimitFilterFieldEnum>;

export type FilterFieldConfigs = {
  startTime: NumberConfig<RatelimitFilterOperator>;
  endTime: NumberConfig<RatelimitFilterOperator>;
  since: StringConfig<RatelimitFilterOperator>;
  identifiers: StringConfig<RatelimitFilterOperator>;
  status: StringConfig<RatelimitFilterOperator>;
};

export type RatelimitFilterUrlValue = Pick<
  FilterValue<RatelimitFilterField, RatelimitFilterOperator>,
  "value" | "operator"
>;
export type RatelimitFilterValue = FilterValue<RatelimitFilterField, RatelimitFilterOperator>;

export type RatelimitQuerySearchParams = {
  startTime?: number | null;
  endTime?: number | null;
  since?: string | null;
  identifiers: RatelimitFilterUrlValue[] | null;
  status: RatelimitFilterUrlValue[] | null;
};
