import {
  rolesFilterOperatorEnum,
  rolesListFilterFieldNames,
} from "@/app/(app)/[workspaceSlug]/authorization/roles/filters.schema";
import { z } from "zod";

const filterItemSchema = z.object({
  operator: rolesFilterOperatorEnum,
  value: z.string(),
});

const baseFilterArraySchema = z.array(filterItemSchema).nullish();

type FilterFieldName = (typeof rolesListFilterFieldNames)[number];

const filterFieldsSchema = rolesListFilterFieldNames.reduce(
  (acc, fieldName) => {
    acc[fieldName] = baseFilterArraySchema;
    return acc;
  },
  {} as Record<FilterFieldName, typeof baseFilterArraySchema>,
);

const baseRolesSchema = z.object(filterFieldsSchema);

const rolesSortByEnum = z.enum(["name", "lastUpdated", "assignedKeys", "assignedPermissions"]);
const rolesSortOrderEnum = z.enum(["asc", "desc"]);

export const rolesQueryPayload = baseRolesSchema.extend({
  page: z.number().int().min(1).optional().default(1),
  limit: z.number().int().min(1).max(100).optional(),
  sortBy: rolesSortByEnum.optional().default("lastUpdated"),
  sortOrder: rolesSortOrderEnum.optional().default("desc"),
});

export type RolesSortField = z.infer<typeof rolesSortByEnum>;
export type RolesSortOrder = z.infer<typeof rolesSortOrderEnum>;
export type RolesQueryPayload = z.infer<typeof rolesQueryPayload>;
