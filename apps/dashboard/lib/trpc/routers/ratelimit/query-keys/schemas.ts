import { z } from "zod";

export const ratelimitNamespace = z.object({
  id: z.string(),
  name: z.string(),
});

export type RatelimitNamespace = z.infer<typeof ratelimitNamespace>;

const Cursor = z.object({
  id: z.string(),
});

export const ratelimitNamespacesResponse = z.object({
  namespaceList: z.array(ratelimitNamespace),
  hasMore: z.boolean(),
  nextCursor: Cursor.optional(),
  total: z.number(),
});

export type RatelimitNamespacesResponse = z.infer<typeof ratelimitNamespacesResponse>;

export const queryRatelimitNamespacesPayload = z.object({
  limit: z.number().min(1).max(18).default(9),
  cursor: Cursor.optional(),
});
