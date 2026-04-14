import { z } from "zod";

export const identitiesQueryPayload = z.object({
  page: z.number().int().min(1).optional().default(1),
  limit: z.number().int().min(1).max(100).optional().default(50),
  search: z.string().optional(),
});

export type IdentitiesQueryPayload = z.infer<typeof identitiesQueryPayload>;
