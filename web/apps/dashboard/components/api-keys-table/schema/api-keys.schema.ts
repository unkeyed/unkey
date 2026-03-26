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

export const apiKeysQueryPayload = z.object({
  keyAuthId: z.string(),
  names: baseFilterArraySchema,
  identities: baseFilterArraySchema,
  keyIds: baseFilterArraySchema,
  tags: baseFilterArraySchema,
  page: z.number().int().min(1).optional().default(1),
  limit: z.number().optional().default(PAGINATION_LIMIT),
  sortBy: apiKeysSortFieldEnum.optional().default("id"),
  sortOrder: z.enum(["asc", "desc"]).optional().default("desc"),
});

export type ApiKeysQueryPayload = z.infer<typeof apiKeysQueryPayload>;
