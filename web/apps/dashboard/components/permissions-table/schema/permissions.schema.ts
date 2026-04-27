import {
  permissionsFilterOperatorEnum,
  permissionsListFilterFieldNames,
} from "@/app/(app)/[workspaceSlug]/authorization/permissions/filters.schema";
import { createPaginatedListQueryPayload } from "@/lib/schemas/paginated-list.schema";
import { z } from "zod";

const permissionsSortByEnum = z.enum([
  "name",
  "slug",
  "totalConnectedRoles",
  "totalConnectedKeys",
  "lastUpdated",
]);

export const permissionsQueryPayload = createPaginatedListQueryPayload({
  operatorEnum: permissionsFilterOperatorEnum,
  filterFieldNames: permissionsListFilterFieldNames,
  sortByEnum: permissionsSortByEnum,
  defaultSortField: "lastUpdated",
});

export type PermissionsSortField = z.infer<typeof permissionsSortByEnum>;
export type PermissionsSortOrder = "asc" | "desc";
export type PermissionsQueryPayload = z.infer<typeof permissionsQueryPayload>;
