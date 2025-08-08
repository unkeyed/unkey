import type {
  FilterValue,
  NumberConfig,
  StringConfig,
} from "@/components/logs/validation/filter.types";
import { createFilterOutputSchema } from "@/components/logs/validation/utils/structured-output-schema-generator";
import { z } from "zod";

// Configuration
export const ratelimitOverviewFilterFieldConfig: FilterFieldConfigs = {
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
export const ratelimitOverviewFilterOperatorEnum = z.enum(["is", "contains"]);
export const ratelimitOverviewFilterFieldEnum = z.enum([
  "startTime",
  "endTime",
  "since",
  "identifiers",
  "status",
]);
export const filterOutputSchema = createFilterOutputSchema(
  ratelimitOverviewFilterFieldEnum,
  ratelimitOverviewFilterOperatorEnum,
  ratelimitOverviewFilterFieldConfig,
);

// Types
export type RatelimitOverviewFilterOperator = z.infer<typeof ratelimitOverviewFilterOperatorEnum>;
export type RatelimitOverviewFilterField = z.infer<typeof ratelimitOverviewFilterFieldEnum>;

export type FilterFieldConfigs = {
  startTime: NumberConfig<RatelimitOverviewFilterOperator>;
  endTime: NumberConfig<RatelimitOverviewFilterOperator>;
  since: StringConfig<RatelimitOverviewFilterOperator>;
  identifiers: StringConfig<RatelimitOverviewFilterOperator>;
  status: StringConfig<RatelimitOverviewFilterOperator>;
};

export type RatelimitOverviewFilterUrlValue = Pick<
  FilterValue<RatelimitOverviewFilterField, RatelimitOverviewFilterOperator>,
  "value" | "operator"
>;
export type RatelimitOverviewFilterValue = FilterValue<
  RatelimitOverviewFilterField,
  RatelimitOverviewFilterOperator
>;

export type RatelimitQuerySearchParams = {
  startTime?: number | null;
  endTime?: number | null;
  since?: string | null;
  identifiers: RatelimitOverviewFilterUrlValue[] | null;
  status: RatelimitOverviewFilterUrlValue[] | null;
};
