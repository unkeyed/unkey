import type { FilterValue, StringConfig } from "@/components/logs/validation/filter.types";
import { parseAsFilterValueArray } from "@/components/logs/validation/utils/nuqs-parsers";
import { createFilterOutputSchema } from "@/components/logs/validation/utils/structured-output-schema-generator";
import { z } from "zod";

const commonStringOperators = ["is", "contains", "startsWith", "endsWith"] as const;

export const rolesFilterOperatorEnum = z.enum(commonStringOperators);
export type RolesFilterOperator = z.infer<typeof rolesFilterOperatorEnum>;

export type FilterFieldConfigs = {
  description: StringConfig<RolesFilterOperator>;
  name: StringConfig<RolesFilterOperator>;
  permissionSlug: StringConfig<RolesFilterOperator>;
  permissionName: StringConfig<RolesFilterOperator>;
  keyId: StringConfig<RolesFilterOperator>;
  keyName: StringConfig<RolesFilterOperator>;
};

export const rolesFilterFieldConfig: FilterFieldConfigs = {
  name: {
    type: "string",
    operators: [...commonStringOperators],
  },
  description: {
    type: "string",
    operators: [...commonStringOperators],
  },
  permissionSlug: {
    type: "string",
    operators: [...commonStringOperators],
  },
  permissionName: {
    type: "string",
    operators: [...commonStringOperators],
  },
  keyId: {
    type: "string",
    operators: [...commonStringOperators],
  },
  keyName: {
    type: "string",
    operators: [...commonStringOperators],
  },
};

const allFilterFieldNames = Object.keys(rolesFilterFieldConfig) as (keyof FilterFieldConfigs)[];

if (allFilterFieldNames.length === 0) {
  throw new Error("rolesFilterFieldConfig must contain at least one field definition.");
}

const [firstFieldName, ...restFieldNames] = allFilterFieldNames;

export const rolesFilterFieldEnum = z.enum([firstFieldName, ...restFieldNames]);

export const rolesListFilterFieldNames = allFilterFieldNames;

export type RolesFilterField = z.infer<typeof rolesFilterFieldEnum>;

export const filterOutputSchema = createFilterOutputSchema(
  rolesFilterFieldEnum,
  rolesFilterOperatorEnum,
  rolesFilterFieldConfig,
);

export type AllOperatorsUrlValue = {
  value: string;
  operator: RolesFilterOperator;
};

export type RolesFilterValue = FilterValue<RolesFilterField, RolesFilterOperator>;

export type RolesQuerySearchParams = {
  [K in RolesFilterField]?: AllOperatorsUrlValue[] | null;
};

export const parseAsAllOperatorsFilterArray = parseAsFilterValueArray<RolesFilterOperator>([
  ...commonStringOperators,
]);
