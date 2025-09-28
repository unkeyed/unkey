import type { FilterValue, StringConfig } from "@/components/logs/validation/filter.types";
import { parseAsFilterValueArray } from "@/components/logs/validation/utils/nuqs-parsers";
import { createFilterOutputSchema } from "@/components/logs/validation/utils/structured-output-schema-generator";
import { z } from "zod";

const commonStringOperators = ["is", "contains", "startsWith", "endsWith"] as const;
export const permissionsFilterOperatorEnum = z.enum(commonStringOperators);
export type PermissionsFilterOperator = z.infer<typeof permissionsFilterOperatorEnum>;

export type FilterFieldConfigs = {
  description: StringConfig<PermissionsFilterOperator>;
  name: StringConfig<PermissionsFilterOperator>;
  slug: StringConfig<PermissionsFilterOperator>;
  roleId: StringConfig<PermissionsFilterOperator>;
  roleName: StringConfig<PermissionsFilterOperator>;
};

export const permissionsFilterFieldConfig: FilterFieldConfigs = {
  name: {
    type: "string",
    operators: [...commonStringOperators],
  },
  description: {
    type: "string",
    operators: [...commonStringOperators],
  },
  slug: {
    type: "string",
    operators: [...commonStringOperators],
  },
  roleId: {
    type: "string",
    operators: [...commonStringOperators],
  },
  roleName: {
    type: "string",
    operators: [...commonStringOperators],
  },
};

const allFilterFieldNames = Object.keys(
  permissionsFilterFieldConfig,
) as (keyof FilterFieldConfigs)[];

if (allFilterFieldNames.length === 0) {
  throw new Error("permissionsFilterFieldConfig must contain at least one field definition.");
}

const [firstFieldName, ...restFieldNames] = allFilterFieldNames;

export const permissionsFilterFieldEnum = z.enum([firstFieldName, ...restFieldNames]);
export const permissionsListFilterFieldNames = allFilterFieldNames;
export type PermissionsFilterField = z.infer<typeof permissionsFilterFieldEnum>;

export const filterOutputSchema = createFilterOutputSchema(
  permissionsFilterFieldEnum,
  permissionsFilterOperatorEnum,
  permissionsFilterFieldConfig,
);

export type AllOperatorsUrlValue = {
  value: string;
  operator: PermissionsFilterOperator;
};

export type PermissionsFilterValue = FilterValue<PermissionsFilterField, PermissionsFilterOperator>;

export type PermissionsQuerySearchParams = {
  [K in PermissionsFilterField]?: AllOperatorsUrlValue[] | null;
};

export const parseAsAllOperatorsFilterArray = parseAsFilterValueArray<PermissionsFilterOperator>([
  ...commonStringOperators,
]);
