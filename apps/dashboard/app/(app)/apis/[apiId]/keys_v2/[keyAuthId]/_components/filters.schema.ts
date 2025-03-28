import type { FilterValue, StringConfig } from "@/components/logs/validation/filter.types";
import { createFilterOutputSchema } from "@/components/logs/validation/utils/structured-output-schema-generator";
import { z } from "zod";

export const keysListFilterFieldConfig: FilterFieldConfigs = {
  keyIds: {
    type: "string",
    operators: ["is", "contains"],
  },
  names: {
    type: "string",
    operators: ["is", "contains", "startsWith", "endsWith"],
  },
  identities: {
    type: "string",
    operators: ["is", "contains", "startsWith", "endsWith"],
  },
};

// Schemas
export const keysListFilterOperatorEnum = z.enum(["is", "contains", "startsWith", "endsWith"]);

export const keysListFilterFieldEnum = z.enum(["keyIds", "names", "identities"]);

export const filterOutputSchema = createFilterOutputSchema(
  keysListFilterFieldEnum,
  keysListFilterOperatorEnum,
  keysListFilterFieldConfig,
);

// Types
export type KeysListFilterOperator = z.infer<typeof keysListFilterOperatorEnum>;

export type KeysListFilterField = z.infer<typeof keysListFilterFieldEnum>;

export type FilterFieldConfigs = {
  keyIds: StringConfig<KeysListFilterOperator>;
  names: StringConfig<KeysListFilterOperator>;
  identities: StringConfig<KeysListFilterOperator>;
};

export type IsOnlyUrlValue = {
  value: string | number;
  operator: "is";
};

export type IsContainsUrlValue = {
  value: string;
  operator: "is" | "contains";
};

export type AllOperatorsUrlValue = {
  value: string;
  operator: "is" | "contains" | "startsWith" | "endsWith";
};

export type KeysListFilterUrlValue = Pick<
  FilterValue<KeysListFilterField, KeysListFilterOperator>,
  "value" | "operator"
>;

export type KeysListFilterValue = FilterValue<KeysListFilterField, KeysListFilterOperator>;

export type KeysQuerySearchParams = {
  keyIds: IsContainsUrlValue[] | null;
  names: AllOperatorsUrlValue[] | null;
  identities: AllOperatorsUrlValue[] | null;
};
