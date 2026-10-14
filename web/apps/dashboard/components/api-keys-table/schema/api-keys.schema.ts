import { keysListFilterOperatorEnum } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/keys/[keyAuthId]/_components/filters.schema";
import { z } from "zod";

const PAGINATION_LIMIT = 50;

const filterItemSchema = z.object({
  operator: keysListFilterOperatorEnum,
  value: z.string(),
});

const baseFilterArraySchema = z.array(filterItemSchema).nullish();

export const apiKeysSortFieldEnum = z.enum(["id", "start", "lastUsedAt"]);
export type ApiKeysSortField = z.infer<typeof apiKeysSortFieldEnum>;

export const apiKeysCursor = z.object({
  pk: z.number().int().positive().safe(),
  id: z.string().min(1).max(256),
  start: z.string().max(256),
  lastUsedAt: z.number().int().nonnegative(),
});
export type ApiKeysCursor = z.infer<typeof apiKeysCursor>;

export const apiKeysQueryPayload = z.object({
  keyAuthId: z.string(),
  names: baseFilterArraySchema,
  identities: baseFilterArraySchema,
  keyIds: baseFilterArraySchema,
  tags: baseFilterArraySchema,
  cursor: apiKeysCursor.nullish(),
  limit: z.number().int().min(1).max(PAGINATION_LIMIT).optional().default(PAGINATION_LIMIT),
  sortBy: apiKeysSortFieldEnum.optional().default("id"),
  sortOrder: z.enum(["asc", "desc"]).optional().default("desc"),
});

export type ApiKeysQueryPayload = z.infer<typeof apiKeysQueryPayload>;
