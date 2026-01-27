// src/features/keys/filters.schema.ts

import type { FilterValue, StringConfig } from "@/components/logs/validation/filter.types";
import { parseAsFilterValueArray } from "@/components/logs/validation/utils/nuqs-parsers";
import { createFilterOutputSchema } from "@/components/logs/validation/utils/structured-output-schema-generator";
import { z } from "zod";

const commonStringOperators = ["is", "contains", "startsWith", "endsWith"] as const;

export const keysListFilterOperatorEnum = z.enum(commonStringOperators);
export type KeysListFilterOperator = z.infer<typeof keysListFilterOperatorEnum>;

export type FilterFieldConfigs = {
  keyIds: StringConfig<KeysListFilterOperator>;
  names: StringConfig<KeysListFilterOperator>;
  identities: StringConfig<KeysListFilterOperator>;
  tags: StringConfig<KeysListFilterOperator>;
};

export const keysListFilterFieldConfig: FilterFieldConfigs = {
  keyIds: {
    type: "string",
    operators: [...commonStringOperators],
  },
  names: {
    type: "string",
    operators: [...commonStringOperators],
  },
  identities: {
    type: "string",
    operators: [...commonStringOperators],
  },
  tags: {
    type: "string",
    operators: [...commonStringOperators],
  },
};

const allFilterFieldNames = Object.keys(keysListFilterFieldConfig) as (keyof FilterFieldConfigs)[];

if (allFilterFieldNames.length === 0) {
  throw new Error("keysListFilterFieldConfig must contain at least one field definition.");
}

export const keysListFilterFieldEnum = z.enum(["keyIds", "names", "identities", "tags"]);

export const keysListFilterFieldNames = allFilterFieldNames;

export type KeysListFilterField = z.infer<typeof keysListFilterFieldEnum>;

export const filterOutputSchema = createFilterOutputSchema(
  keysListFilterFieldEnum,
  keysListFilterOperatorEnum,
  keysListFilterFieldConfig,
);

export type AllOperatorsUrlValue = {
  value: string;
  operator: KeysListFilterOperator;
};

export type KeysListFilterValue = FilterValue<KeysListFilterField, KeysListFilterOperator>;

export type KeysQuerySearchParams = {
  [K in KeysListFilterField]?: AllOperatorsUrlValue[] | null;
};

export const parseAsAllOperatorsFilterArray = parseAsFilterValueArray<KeysListFilterOperator>([
  ...commonStringOperators,
]);
