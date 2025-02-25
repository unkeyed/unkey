import type {
  FilterValue,
  NumberConfig,
  StringConfig,
} from "@/components/logs/validation/filter.types";
import { z } from "zod";

// Configuration
export const ratelimitListFilterFieldConfig: FilterFieldConfigs = {
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
};

// Schemas
export const ratelimitListFilterOperatorEnum = z.enum(["is", "contains"]);
export const ratelimitListFilterFieldEnum = z.enum(["startTime", "endTime", "since"]);

// Types
export type RatelimitListFilterOperator = z.infer<typeof ratelimitListFilterOperatorEnum>;
export type RatelimitListFilterField = z.infer<typeof ratelimitListFilterFieldEnum>;

export type FilterFieldConfigs = {
  startTime: NumberConfig<RatelimitListFilterOperator>;
  endTime: NumberConfig<RatelimitListFilterOperator>;
  since: StringConfig<RatelimitListFilterOperator>;
};

export type RatelimitListFilterUrlValue = Pick<
  FilterValue<RatelimitListFilterField, RatelimitListFilterOperator>,
  "value" | "operator"
>;
export type RatelimitListFilterValue = FilterValue<
  RatelimitListFilterField,
  RatelimitListFilterOperator
>;

export type RatelimitQuerySearchParams = {
  startTime?: number | null;
  endTime?: number | null;
  since?: string | null;
};
