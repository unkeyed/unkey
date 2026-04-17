import { z } from "zod";

export const identitiesSortFields = ["externalId", "createdAt"] as const;
export type IdentitiesSortField = (typeof identitiesSortFields)[number];

export const identitiesQueryPayload = z.object({
  page: z.number().int().min(1).optional().default(1),
  limit: z.number().int().min(1).max(100).optional().default(50),
  search: z.string().optional(),
  sortBy: z.enum(identitiesSortFields).optional().default("createdAt"),
  sortOrder: z.enum(["asc", "desc"]).optional().default("desc"),
});

export type IdentitiesQueryPayload = z.infer<typeof identitiesQueryPayload>;
