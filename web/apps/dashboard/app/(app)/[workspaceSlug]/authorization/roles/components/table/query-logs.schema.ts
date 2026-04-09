import { z } from "zod";
import { rolesFilterOperatorEnum, rolesListFilterFieldNames } from "../../filters.schema";

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

export const rolesQueryPayload = baseRolesSchema.extend({
  page: z.number().int().min(1).optional().default(1),
  limit: z.number().int().min(1).max(100).optional(),
  sortBy: z
    .enum(["name", "lastUpdated", "assignedKeys", "assignedPermissions"])
    .optional()
    .default("lastUpdated"),
  sortOrder: z.enum(["asc", "desc"]).optional().default("desc"),
});

export type RolesSortField = "name" | "lastUpdated" | "assignedKeys" | "assignedPermissions";
export type RolesSortOrder = "asc" | "desc";
export type RolesQueryPayload = z.infer<typeof rolesQueryPayload>;
