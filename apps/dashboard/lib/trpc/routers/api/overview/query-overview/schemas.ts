import { z } from "zod";

export const apiOverview = z.object({
  id: z.string(),
  name: z.string(),
  keyspaceId: z.string().nullable(),
  // For backward compatibility
  keys: z.array(
    z.object({
      count: z.number(),
    }),
  ),
});

export type ApiOverview = z.infer<typeof apiOverview>;

const Cursor = z.object({
  id: z.string(),
});

export const apisOverviewResponse = z.object({
  apiList: z.array(apiOverview),
  hasMore: z.boolean(),
  nextCursor: Cursor.optional(),
  total: z.number(),
});

export type ApisOverviewResponse = z.infer<typeof apisOverviewResponse>;

export const queryApisOverviewPayload = z.object({
  limit: z.number().min(1).max(18).default(9),
  cursor: Cursor.optional(),
});
