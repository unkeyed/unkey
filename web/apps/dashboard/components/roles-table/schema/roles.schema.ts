import { z } from "zod";
import {
  rolesFilterOperatorEnum,
  rolesListFilterFieldNames,
} from "@/app/(app)/[workspaceSlug]/authorization/roles/filters.schema";
import { createPaginatedListQueryPayload } from "@/lib/schemas/paginated-list.schema";

const rolesSortByEnum = z.enum(["name", "lastUpdated", "assignedKeys", "assignedPermissions"]);

export const rolesQueryPayload = createPaginatedListQueryPayload({
  operatorEnum: rolesFilterOperatorEnum,
  filterFieldNames: rolesListFilterFieldNames,
  sortByEnum: rolesSortByEnum,
  defaultSortField: "lastUpdated",
});

export type RolesSortField = z.infer<typeof rolesSortByEnum>;
export type RolesSortOrder = "asc" | "desc";
export type RolesQueryPayload = z.infer<typeof rolesQueryPayload>;
