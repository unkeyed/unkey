import {
  filterOutputSchema,
  permissionsFilterFieldConfig,
} from "@/app/(app)/[workspaceSlug]/authorization/permissions/filters.schema";
import type { SearchSpec } from "../spec";
import { NAME_RULES } from "./roles";

export const permissionsSearchSpec: SearchSpec = {
  subject: "permissions",
  outputSchema: filterOutputSchema,
  config: permissionsFilterFieldConfig,
  fields: {
    name: "the readable name of the permission itself",
    slug: "the machine-readable slug of the permission, such as api.read",
    description: "the sentence explaining what the permission allows",
    roleName: "the name of a role that grants this permission",
    roleId: "the identifier of a role that grants this permission",
  },
  rules: NAME_RULES,
  examples: [
    {
      query: "find admin and user permissions",
      result: [
        {
          field: "name",
          filters: [
            { operator: "contains", value: "admin" },
            { operator: "contains", value: "user" },
          ],
        },
      ],
    },
    {
      query: "permissions with api.read and api.write slugs",
      result: [
        {
          field: "slug",
          filters: [
            { operator: "is", value: "api.read" },
            { operator: "is", value: "api.write" },
          ],
        },
      ],
    },
    {
      query: "show permissions ending with .create",
      result: [{ field: "slug", filters: [{ operator: "endsWith", value: ".create" }] }],
    },
    {
      query: "permissions for database access",
      result: [{ field: "description", filters: [{ operator: "contains", value: "database" }] }],
    },
    {
      query: "permissions assigned to admin role",
      result: [{ field: "roleName", filters: [{ operator: "contains", value: "admin" }] }],
    },
    {
      query: "find permissions for role_123",
      result: [{ field: "roleId", filters: [{ operator: "is", value: "role_123" }] }],
    },
    {
      query: "admin permissions with database access",
      result: [
        { field: "name", filters: [{ operator: "contains", value: "admin" }] },
        { field: "description", filters: [{ operator: "contains", value: "database" }] },
      ],
    },
    {
      query: "list every permission",
      result: [],
      note: "nothing is named, so nothing is filtered",
    },
  ],
};
