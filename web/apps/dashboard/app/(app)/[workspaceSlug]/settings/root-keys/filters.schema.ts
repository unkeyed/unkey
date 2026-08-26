import type { FilterValue, StringConfig } from "@/components/logs/validation/filter.types";
import { z } from "zod";

const commonStringOperators = ["is", "contains"] as const;

export const rootKeysFilterOperatorEnum = z.enum(commonStringOperators);
export type RootKeysFilterOperator = z.infer<typeof rootKeysFilterOperatorEnum>;

export type FilterFieldConfigs = {
  name: StringConfig<RootKeysFilterOperator>;
};

export const rootKeysFilterFieldConfig: FilterFieldConfigs = {
  name: {
    type: "string",
    operators: [...commonStringOperators],
  },
};

export const rootKeysListFilterFieldNames = ["name"] as const;

export type RootKeysFilterField = (typeof rootKeysListFilterFieldNames)[number];

export type RootKeysFilterValue = FilterValue<RootKeysFilterField, RootKeysFilterOperator>;
