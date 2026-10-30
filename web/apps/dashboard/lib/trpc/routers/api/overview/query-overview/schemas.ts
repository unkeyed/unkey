import { GRID_PAGE_SIZE, SEARCH_LENGTH_MAX } from "@/lib/list-limits";
import { z } from "zod";

export const apiOverview = z.object({
  id: z.string(),
  name: z.string(),
  keyspaceId: z.string().nullable(),
  keyCount: z.number(),
});

export type ApiOverview = z.infer<typeof apiOverview>;

const Cursor = z.object({
  id: z.string(),
});

export const apisOverviewResponse = z.object({
  apiList: z.array(apiOverview),
  hasMore: z.boolean(),
  nextCursor: Cursor.optional(),
});

export type ApisOverviewResponse = z.infer<typeof apisOverviewResponse>;

export const queryApisOverviewPayload = z.object({
  limit: z.number().min(1).max(GRID_PAGE_SIZE).prefault(GRID_PAGE_SIZE),
  cursor: Cursor.optional(),
  search: z.string().trim().max(SEARCH_LENGTH_MAX).optional(),
});
