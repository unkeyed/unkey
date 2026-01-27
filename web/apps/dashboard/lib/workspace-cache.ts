import { db } from "@/lib/db";
import { unstable_cache } from "next/cache";

/**
 * Server-side cached workspace fetcher that maintains consistency with client-side caching
 * Uses Next.js unstable_cache with 10-minute TTL to match server-side cache strategy
 */
export const getCachedWorkspace = (orgId: string) =>
  unstable_cache(
    async () => {
      const workspace = await db.query.workspaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
        with: {
          quotas: true,
        },
      });

      return workspace;
    },
    [`workspace-${orgId}`], // org-specific cache key for better granularity
    {
      revalidate: 600, // 10 minutes to match server-side cache TTL
      tags: [`workspace-${orgId}`, "workspace"], // both specific and general tags
    },
  );

/**
 * Cache invalidation helper for when workspace data changes
 * Can invalidate specific org's workspace or all workspaces
 */
export const invalidateWorkspaceCache = async (orgId?: string) => {
  const { revalidateTag } = await import("next/cache");

  if (orgId) {
    // Invalidate specific org's workspace cache
    revalidateTag(`workspace-${orgId}`, "max");
  } else {
    // Invalidate all workspace caches
    revalidateTag("workspace", "max");
  }
};

/**
 * Helper to get orgId from workspaceId for cache invalidation
 * This is needed when we only have workspaceId but need to invalidate by orgId
 */
export const invalidateWorkspaceCacheById = async (workspaceId: string) => {
  // First fetch the workspace to get the orgId
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { eq }) => eq(table.id, workspaceId),
    columns: {
      orgId: true,
    },
  });

  if (workspace) {
    await invalidateWorkspaceCache(workspace.orgId);
  }
};

/**
 * Batch invalidate multiple workspace caches by orgId
 */
export const invalidateMultipleWorkspaceCaches = async (orgIds: string[]) => {
  const { revalidateTag } = await import("next/cache");

  for (const orgId of orgIds) {
    revalidateTag(`workspace-${orgId}`, "max");
  }
};
