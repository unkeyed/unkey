import {
  filterOutputSchema,
  keysListFilterFieldConfig,
} from "@/app/(app)/[workspaceSlug]/apis/[apiId]/keys/[keyAuthId]/_components/filters.schema";
import type { SearchSpec } from "../spec";
import { NAME_RULES } from "./roles";

export const KEY_FIELDS = {
  keyIds: "the identifier of a key",
  names: "the name someone gave a key",
  identities: "the external identity a key belongs to, such as a user or a tenant",
  tags: "a label attached to a key",
} as const;

export const keysListSearchSpec: SearchSpec = {
  subject: "API keys",
  outputSchema: filterOutputSchema,
  config: keysListFilterFieldConfig,
  fields: { ...KEY_FIELDS },
  rules: NAME_RULES,
  examples: [
    {
      query: "find test keys",
      result: [{ field: "names", filters: [{ operator: "contains", value: "test" }] }],
      note: "an unqualified name is a substring search",
    },
    {
      query: "keys whose name starts with prod",
      result: [{ field: "names", filters: [{ operator: "startsWith", value: "prod" }] }],
    },
    {
      query: "keys tagged internal",
      result: [{ field: "tags", filters: [{ operator: "is", value: "internal" }] }],
    },
    {
      query: "keys for identity user_9fj3ks01",
      result: [{ field: "identities", filters: [{ operator: "is", value: "user_9fj3ks01" }] }],
    },
    {
      query: "key key_3ZZabc",
      result: [{ field: "keyIds", filters: [{ operator: "is", value: "key_3ZZabc" }] }],
    },
    { query: "show all keys", result: [], note: "nothing is named, so nothing is filtered" },
  ],
};
