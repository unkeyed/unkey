import type { FilterValue, StringConfig } from "@/components/logs/validation/filter.types";
import { parseAsFilterValueArray } from "@/components/logs/validation/utils/nuqs-parsers";
import { z } from "zod";

const commonStringOperators = ["contains"] as const;
export const projectsFilterOperatorEnum = z.enum(commonStringOperators);
type ProjectsFilterOperator = z.infer<typeof projectsFilterOperatorEnum>;

export type FilterFieldConfigs = {
  query: StringConfig<ProjectsFilterOperator>;
};

export const projectsFilterFieldConfig: FilterFieldConfigs = {
  query: {
    type: "string",
    operators: [...commonStringOperators],
  },
};

const allFilterFieldNames = Object.keys(projectsFilterFieldConfig) as (keyof FilterFieldConfigs)[];
if (allFilterFieldNames.length === 0) {
  throw new Error("projectsFilterFieldConfig must contain at least one field definition.");
}

const [firstFieldName, ...restFieldNames] = allFilterFieldNames;
const projectsFilterFieldEnum = z.enum([firstFieldName, ...restFieldNames]);
export const projectsListFilterFieldNames = allFilterFieldNames;
export type ProjectsFilterField = z.infer<typeof projectsFilterFieldEnum>;

export type AllOperatorsUrlValue = {
  value: string;
  operator: ProjectsFilterOperator;
};

export type ProjectsFilterValue = FilterValue<ProjectsFilterField, ProjectsFilterOperator>;

export type ProjectsQuerySearchParams = {
  [K in ProjectsFilterField]?: AllOperatorsUrlValue[] | null;
};

export const parseAsAllOperatorsFilterArray = parseAsFilterValueArray<ProjectsFilterOperator>([
  ...commonStringOperators,
]);
