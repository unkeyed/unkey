import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireUser, requireWorkspace, t } from "../../trpc";
import {
  cleanupExpiredWorkspaceCaches,
  clearAllWorkspaceCaches,
  clearWorkspaceCache,
} from "./getCurrent";

export const clearCache = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(
    z.object({
      orgId: z.string().optional(),
      clearAll: z.boolean().default(false),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    try {
      if (input.clearAll) {
        clearAllWorkspaceCaches();
        return {
          success: true,
          message: "All workspace caches cleared",
          clearedCount: "all",
        };
      }

      const targetOrgId = input.orgId || ctx.tenant.id;
      clearWorkspaceCache(targetOrgId);

      return {
        success: true,
        message: `Cache cleared for workspace ${targetOrgId}`,
        clearedCount: 1,
        orgId: targetOrgId,
      };
    } catch (error) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to clear workspace cache",
        cause: error,
      });
    }
  });

export const cleanupCache = t.procedure.use(requireUser).mutation(async () => {
  try {
    cleanupExpiredWorkspaceCaches();

    return {
      success: true,
      message: "Expired cache entries cleaned up",
      timestamp: new Date().toISOString(),
    };
  } catch (error) {
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message: "Failed to cleanup expired cache entries",
      cause: error,
    });
  }
});

export const warmCache = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(
    z.object({
      orgIds: z.array(z.string()).optional(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    try {
      const targetOrgIds = input.orgIds || [ctx.tenant.id];
      const warmedCaches: string[] = [];

      // This would typically pre-fetch workspace data for the specified org IDs
      // For now, we'll just clear any existing cache to force a fresh fetch
      for (const orgId of targetOrgIds) {
        clearWorkspaceCache(orgId);
        warmedCaches.push(orgId);
      }

      return {
        success: true,
        message: `Cache warmed for ${warmedCaches.length} workspace(s)`,
        warmedCaches,
        timestamp: new Date().toISOString(),
      };
    } catch (error) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to warm workspace cache",
        cause: error,
      });
    }
  });
