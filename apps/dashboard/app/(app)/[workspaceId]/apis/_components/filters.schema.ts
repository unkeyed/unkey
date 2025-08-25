import type {
  FilterValue,
  NumberConfig,
  StringConfig,
} from "@/components/logs/validation/filter.types";
import { z } from "zod";

// Configuration
export const apiListFilterFieldConfig: FilterFieldConfigs = {
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
export const apiListFilterOperatorEnum = z.enum(["is", "contains"]);
export const apiListFilterFieldEnum = z.enum(["startTime", "endTime", "since"]);

// Types
export type ApiListFilterOperator = z.infer<typeof apiListFilterOperatorEnum>;
export type ApiListFilterField = z.infer<typeof apiListFilterFieldEnum>;

export type FilterFieldConfigs = {
  startTime: NumberConfig<ApiListFilterOperator>;
  endTime: NumberConfig<ApiListFilterOperator>;
  since: StringConfig<ApiListFilterOperator>;
};

export type ApiListFilterUrlValue = Pick<
  FilterValue<ApiListFilterField, ApiListFilterOperator>,
  "value" | "operator"
>;
export type ApiListFilterValue = FilterValue<ApiListFilterField, ApiListFilterOperator>;

export type ApiListQuerySearchParams = {
  startTime?: number | null;
  endTime?: number | null;
  since?: string | null;
};
