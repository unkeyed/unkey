import {
  type FilterFieldConfigs,
  permissionsFilterOperatorEnum,
  permissionsListFilterFieldNames,
} from "@/app/(app)/[workspaceSlug]/authorization/permissions/filters.schema";
import { z } from "zod";

const filterItemSchema = z.object({
  operator: permissionsFilterOperatorEnum,
  value: z.string(),
});

const baseFilterArraySchema = z.array(filterItemSchema).nullish();

type FilterFieldName = keyof FilterFieldConfigs;

const filterFieldsSchema = permissionsListFilterFieldNames.reduce(
  (acc, fieldName) => {
    acc[fieldName] = baseFilterArraySchema;
    return acc;
  },
  {} as Record<FilterFieldName, typeof baseFilterArraySchema>,
);

const basePermissionsSchema = z.object(filterFieldsSchema);

export const permissionsQueryPayload = basePermissionsSchema.extend({
  page: z.number().int().min(1).optional().default(1),
  limit: z.number().int().min(1).max(100).optional(),
  sortBy: z
    .enum(["name", "slug", "totalConnectedRoles", "totalConnectedKeys", "lastUpdated"])
    .optional()
    .default("lastUpdated"),
  sortOrder: z.enum(["asc", "desc"]).optional().default("desc"),
});

export type PermissionsSortField =
  | "name"
  | "slug"
  | "totalConnectedRoles"
  | "totalConnectedKeys"
  | "lastUpdated";
export type PermissionsSortOrder = "asc" | "desc";
export type PermissionsQueryPayload = z.infer<typeof permissionsQueryPayload>;
