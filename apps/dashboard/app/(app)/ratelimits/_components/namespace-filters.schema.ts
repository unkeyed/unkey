import { parseAsFilterValueArray } from "@/components/logs/validation/utils/nuqs-parsers";
import { createFilterOutputSchema } from "@/components/logs/validation/utils/structured-output-schema-generator";
import type {
  FilterValue,
  NumberConfig,
  StringConfig,
} from "@unkey/ui/src/validation/filter.types";
import { z } from "zod";

export const commonStringOperators = ["contains", "is"] as const;
export const namespaceFilterOperatorEnum = z.enum(commonStringOperators);
export type NamespaceListFilterOperator = z.infer<typeof namespaceFilterOperatorEnum>;

export type NamespaceListFilterFieldConfigs = {
  query: StringConfig<NamespaceListFilterOperator>;
  since: StringConfig<NamespaceListFilterOperator>;
  startTime: NumberConfig<NamespaceListFilterOperator>;
  endTime: NumberConfig<NamespaceListFilterOperator>;
};

export const namespaceListFilterFieldConfig: NamespaceListFilterFieldConfigs = {
  query: {
    type: "string",
    operators: ["contains"],
  },
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

const allFilterFieldNames = Object.keys(
  namespaceListFilterFieldConfig,
) as (keyof NamespaceListFilterFieldConfigs)[];
if (allFilterFieldNames.length === 0) {
  throw new Error("namespaceFilterFieldConfig must contain at least one field definition.");
}

const [firstFieldName, ...restFieldNames] = allFilterFieldNames;
export const namespaceFilterFieldEnum = z.enum([firstFieldName, ...restFieldNames]);
export const namespaceListFilterFieldNames = allFilterFieldNames;
export type NamespaceListFilterField = z.infer<typeof namespaceFilterFieldEnum>;

export const filterOutputSchema = createFilterOutputSchema(
  namespaceFilterFieldEnum,
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
