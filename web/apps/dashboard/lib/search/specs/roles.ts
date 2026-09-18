import {
  filterOutputSchema,
  rolesFilterFieldConfig,
} from "@/app/(app)/[workspaceSlug]/authorization/roles/filters.schema";
import type { SearchSpec } from "../spec";

export const NAME_RULES = [
  "A name or description is searched by substring unless the query asks for an exact match, so prefer contains.",
  "A machine-readable slug or an identifier is matched exactly, so prefer is.",
];

export const rolesSearchSpec: SearchSpec = {
  subject: "roles",
  outputSchema: filterOutputSchema,
  config: rolesFilterFieldConfig,
  fields: {
    name: "the name of the role itself",
    description: "the sentence explaining what the role is for",
    permissionSlug: "the machine-readable slug of a permission the role grants, such as api.read",
    permissionName: "the readable name of a permission the role grants",
    keyId: "the identifier of a key the role is attached to",
    keyName: "the name of a key the role is attached to",
  },
  rules: NAME_RULES,
  examples: [
    {
      query: "find admin and moderator roles",
      result: [
        {
          field: "name",
          filters: [
            { operator: "contains", value: "admin" },
            { operator: "contains", value: "moderator" },
          ],
        },
      ],
    },
    {
      query: "show roles starting with user_",
      result: [{ field: "name", filters: [{ operator: "startsWith", value: "user_" }] }],
    },
    {
      query: "roles for database access",
      result: [{ field: "description", filters: [{ operator: "contains", value: "database" }] }],
    },
    {
      query: "roles with api.read and api.write permissions",
      result: [
        {
          field: "permissionSlug",
          filters: [
            { operator: "is", value: "api.read" },
            { operator: "is", value: "api.write" },
          ],
        },
      ],
      note: "a slug is matched exactly",
    },
    {
      query: "show roles with admin permissions",
      result: [{ field: "permissionName", filters: [{ operator: "contains", value: "admin" }] }],
      note: "a readable permission name is matched by substring",
    },
    {
      query: "roles for production keys",
      result: [{ field: "keyName", filters: [{ operator: "contains", value: "production" }] }],
    },
    {
      query: "admin roles with database permissions and user keys",
      result: [
        { field: "name", filters: [{ operator: "contains", value: "admin" }] },
        { field: "permissionName", filters: [{ operator: "contains", value: "database" }] },
        { field: "keyName", filters: [{ operator: "contains", value: "user" }] },
      ],
    },
    {
      query: "show me all roles",
      result: [],
      note: "nothing is named, so nothing is filtered",
    },
  ],
};
