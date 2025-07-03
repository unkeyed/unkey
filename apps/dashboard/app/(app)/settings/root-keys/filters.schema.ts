import type { FilterValue, StringConfig } from "@/components/logs/validation/filter.types";
import { parseAsFilterValueArray } from "@/components/logs/validation/utils/nuqs-parsers";
import { createFilterOutputSchema } from "@/components/logs/validation/utils/structured-output-schema-generator";
import { z } from "zod";

const commonStringOperators = ["is", "contains"] as const;

export const rootKeysFilterOperatorEnum = z.enum(commonStringOperators);
export type RootKeysFilterOperator = z.infer<typeof rootKeysFilterOperatorEnum>;

export type FilterFieldConfigs = {
  name: StringConfig<RootKeysFilterOperator>;
  start: StringConfig<RootKeysFilterOperator>;
  permission: StringConfig<RootKeysFilterOperator>;
};

export const rootKeysFilterFieldConfig: FilterFieldConfigs = {
  name: {
    type: "string",
    operators: [...commonStringOperators],
  },
  start: {
    type: "string",
    operators: [...commonStringOperators],
  },
  permission: {
    type: "string",
    operators: ["contains"],
  },
};

const allFilterFieldNames = Object.keys(rootKeysFilterFieldConfig) as (keyof FilterFieldConfigs)[];

if (allFilterFieldNames.length === 0) {
  throw new Error("rootKeysFilterFieldConfig must contain at least one field definition.");
}

const [firstFieldName, ...restFieldNames] = allFilterFieldNames;

export const rootKeysFilterFieldEnum = z.enum([firstFieldName, ...restFieldNames]);
export const rootKeysListFilterFieldNames = allFilterFieldNames;
export type RootKeysFilterField = z.infer<typeof rootKeysFilterFieldEnum>;

export const filterOutputSchema = createFilterOutputSchema(
  rootKeysFilterFieldEnum,
  rootKeysFilterOperatorEnum,
  rootKeysFilterFieldConfig,
);

export type AllOperatorsUrlValue = {
  value: string;
  operator: RootKeysFilterOperator;
};

export type RootKeysFilterValue = FilterValue<RootKeysFilterField, RootKeysFilterOperator>;

export type RootKeysQuerySearchParams = {
  [K in RootKeysFilterField]?: AllOperatorsUrlValue[] | null;
};

export const parseAsAllOperatorsFilterArray = parseAsFilterValueArray<RootKeysFilterOperator>([
  ...commonStringOperators,
]);
