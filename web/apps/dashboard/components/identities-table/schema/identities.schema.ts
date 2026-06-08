import { z } from "zod";

export const identitiesSortFields = [
  "externalId",
  "createdAt",
  "keyCount",
  "ratelimitCount",
  "lastUsed",
] as const;
export type IdentitiesSortField = (typeof identitiesSortFields)[number];

export const identitiesFilterOperators = ["is", "contains", "startsWith", "endsWith"] as const;
export type IdentitiesFilterOperator = (typeof identitiesFilterOperators)[number];

const filterItemSchema = z.object({
  operator: z.enum(identitiesFilterOperators),
  value: z.string(),
});

export const identitiesQueryPayload = z.object({
  page: z.number().int().min(1).optional().default(1),
  limit: z.number().int().min(1).max(100).optional().default(50),
  search: z.string().optional(),
  externalId: z.array(filterItemSchema).nullish(),
  lastUsedStart: z.number().optional(),
  lastUsedEnd: z.number().optional(),
  lastUsedSince: z.string().optional(),
  sortBy: z.enum(identitiesSortFields).optional().default("createdAt"),
  sortOrder: z.enum(["asc", "desc"]).optional().default("desc"),
});

export type IdentitiesQueryPayload = z.infer<typeof identitiesQueryPayload>;
