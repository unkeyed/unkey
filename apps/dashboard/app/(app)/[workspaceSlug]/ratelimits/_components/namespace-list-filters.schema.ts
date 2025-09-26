import { parseAsFilterValueArray } from "@/components/logs/validation/utils/nuqs-parsers";
import { createFilterOutputSchema } from "@/components/logs/validation/utils/structured-output-schema-generator";
import type { FilterValue, StringConfig } from "@unkey/ui/src/validation/filter.types";
import { z } from "zod";

export const commonStringOperators = ["contains"] as const;
export const namespaceFilterOperatorEnum = z.enum(commonStringOperators);
export type NamespaceListFilterOperator = z.infer<typeof namespaceFilterOperatorEnum>;

export type NamespaceListFilterFieldConfigs = {
  query: StringConfig<NamespaceListFilterOperator>;
};

export const namespaceListFilterFieldConfig: NamespaceListFilterFieldConfigs = {
  query: {
    type: "string",
    operators: ["contains"],
  },
};

const allFilterFieldNames = Object.keys(
  namespaceListFilterFieldConfig,
) as (keyof NamespaceListFilterFieldConfigs)[];
if (allFilterFieldNames.length === 0) {
  throw new Error("namespaceListFilterFieldConfig must contain at least one field definition.");
}

const [firstFieldName, ...restFieldNames] = allFilterFieldNames;
export const namespaceListFilterFieldEnum = z.enum([firstFieldName, ...restFieldNames]);
export const namespaceListFilterFieldNames = allFilterFieldNames;
export type NamespaceListFilterField = z.infer<typeof namespaceListFilterFieldEnum>;

export const filterOutputSchema = createFilterOutputSchema(
  namespaceListFilterFieldEnum,
  namespaceFilterOperatorEnum,
  namespaceListFilterFieldConfig,
);

export type NamespaceListFilterUrlValue = {
  value: string;
  operator: NamespaceListFilterOperator;
};

export type NamespaceListFilterValue = FilterValue<
  NamespaceListFilterField,
  NamespaceListFilterOperator
>;

export type NamespaceListQuerySearchParams = {
  [K in NamespaceListFilterField]?: K extends "startTime" | "endTime"
    ? number | null
    : K extends "since"
      ? string | null
      : NamespaceListFilterUrlValue[] | null;
};

export const parseAsAllOperatorsFilterArray = parseAsFilterValueArray<NamespaceListFilterOperator>([
  ...commonStringOperators,
]);
