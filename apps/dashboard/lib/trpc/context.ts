import type { inferAsyncReturnType } from "@trpc/server";
import type { FetchCreateContextFnOptions } from "@trpc/server/adapters/fetch";

import { getAuth } from "../auth/get-auth";
import { db } from "../db";

type WorkspaceQueryResult = Awaited<ReturnType<typeof db.query.workspaces.findFirst>>;
type Workspace = WorkspaceQueryResult;

type CacheEntry = {
  workspace: Workspace;
  timestamp: number;
};

const workspaceCache = new Map<string, CacheEntry>();

const CACHE_TTL = 5 * 60 * 1000;

export async function createContext({ req }: FetchCreateContextFnOptions) {
  const { userId, orgId } = await getAuth(req as any);

  let ws: Workspace | undefined;

  if (orgId) {
    const cacheKey = orgId;
    const now = Date.now();
    const cachedEntry = workspaceCache.get(cacheKey);

    if (cachedEntry && now - cachedEntry.timestamp < CACHE_TTL) {
      // --- Cache Hit ---
      ws = cachedEntry.workspace;
    } else {
      // --- Cache Miss or Expired ---

      ws = await db.query.workspaces.findFirst({
        where: (table, { eq, and, isNull }) =>
          and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
        with: {
          apis: {
            where: (table, { isNull }) => isNull(table.deletedAtM),
          },
          quotas: true,
          ratelimitNamespaces: {
            where: (table, { isNull }) => isNull(table.deletedAtM),
            columns: {
              id: true,
              name: true,
            },
          },
        },
      });

      workspaceCache.set(cacheKey, { workspace: ws, timestamp: now });
    }
  } else {
    // No orgId provided, so no workspace to fetch or cache
    ws = undefined;
  }

  return {
    req,
    audit: {
      userAgent: req.headers.get("user-agent") ?? undefined,
      location: req.headers.get("x-forwarded-for") ?? process.env.VERCEL_REGION ?? "unknown",
    },
    user: userId ? { id: userId } : null,
    workspace: ws, // Use the cached or newly fetched workspace data
    tenant: orgId
      ? {
          id: orgId,
        }
      : null,
  };
}

export type Context = inferAsyncReturnType<typeof createContext>;

export function clearWorkspaceCache() {
  workspaceCache.clear();
}

export function invalidateWorkspaceCache(orgId: string) {
  workspaceCache.delete(orgId);
}
