import {
  filterOutputSchema,
  rootKeysFilterFieldConfig,
} from "@/app/(app)/[workspaceSlug]/settings/root-keys/filters.schema";
import type { SearchSpec } from "../spec";
import { NAME_RULES } from "./roles";

export const rootKeysSearchSpec: SearchSpec = {
  subject: "root keys",
  outputSchema: filterOutputSchema,
  config: rootKeysFilterFieldConfig,
  fields: {
    name: "the name someone gave the root key",
    start: "the opening characters a root key begins with, such as unkey_3ZZ",
    permission: "a permission the root key carries, such as api.read_api",
  },
  rules: [
    ...NAME_RULES,
    "A prefix the key begins with is matched by contains unless the query says the whole prefix exactly.",
  ],
  examples: [
    {
      query: "find admin and production keys",
      result: [
        {
          field: "name",
          filters: [
            { operator: "contains", value: "admin" },
            { operator: "contains", value: "production" },
          ],
        },
      ],
    },
    {
      query: "show keys named exactly 'master_key'",
      result: [{ field: "name", filters: [{ operator: "is", value: "master_key" }] }],
    },
    {
      query: "keys starting with sk_",
      result: [{ field: "start", filters: [{ operator: "contains", value: "sk_" }] }],
    },
    {
      query: "keys with exact start sk_1234",
      result: [{ field: "start", filters: [{ operator: "is", value: "sk_1234" }] }],
    },
    {
      query: "find keys with api.create and api.delete",
      result: [
        {
          field: "permission",
          filters: [
            { operator: "contains", value: "api.create" },
            { operator: "contains", value: "api.delete" },
          ],
        },
      ],
    },
    {
      query: "admin keys starting with sk_ with delete permissions",
      result: [
        { field: "name", filters: [{ operator: "contains", value: "admin" }] },
        { field: "start", filters: [{ operator: "contains", value: "sk_" }] },
        { field: "permission", filters: [{ operator: "contains", value: "delete" }] },
      ],
    },
    { query: "show all root keys", result: [], note: "nothing is named, so nothing is filtered" },
  ],
};
