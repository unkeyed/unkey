import {
  type BaseFieldConfig,
  COMMON_STRING_OPERATORS,
  createFilterSchema,
} from "@/lib/filter-builders";
import type { z } from "zod";

const PERMISSIONS_FIELDS = [
  "name",
  "description",
  "slug",
  "roleId",
  "roleName",
] as const;

const PERMISSIONS_FIELD_CONFIGS = Object.fromEntries(
  PERMISSIONS_FIELDS.map((field) => [
    field,
    { type: "string", operators: COMMON_STRING_OPERATORS },
  ])
) as unknown as Record<
  (typeof PERMISSIONS_FIELDS)[number],
  BaseFieldConfig<readonly (typeof COMMON_STRING_OPERATORS)[number][]>
>;

export const permissionsFilter = createFilterSchema(
  COMMON_STRING_OPERATORS,
  PERMISSIONS_FIELDS,
  PERMISSIONS_FIELD_CONFIGS
);

// Direct exports - no destructuring needed
export const permissionsFilterFieldConfig = permissionsFilter.filterFieldConfig;
export const permissionsListFilterFieldNames = permissionsFilter.fieldNames;
export const filterOutputSchema = permissionsFilter.filterOutputSchema;
export const parseAsAllOperatorsFilterArray =
  permissionsFilter.parseAsFilterArray;
export const queryParamsPayload = permissionsFilter.queryParamsPayload;

// Type exports
export type PermissionsFilterOperator = z.infer<
  typeof permissionsFilter.operatorEnum
>;
export type PermissionsFilterField = z.infer<
  typeof permissionsFilter.fieldEnum
>;
export type PermissionsFilterValue = typeof permissionsFilter.types.FilterValue;
export type AllOperatorsUrlValue = typeof permissionsFilter.types.UrlValue;
export type PermissionsQuerySearchParams =
  typeof permissionsFilter.types.QuerySearchParams;
