import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { requireUser, t } from "../../trpc";

// Type definition for workspace data with quotas (inferred from Drizzle query)
type WorkspaceWithQuotas = NonNullable<
  Awaited<
    ReturnType<
      typeof db.query.workspaces.findFirst<{
        with: { quotas: true };
      }>
    >
  >
>;

// In-memory cache for workspace data
const workspaceCache = new Map<string, { data: WorkspaceWithQuotas; timestamp: number }>();
const CACHE_TTL = 1000 * 60 * 10; // 10 minutes server-side cache

export const getCurrentWorkspace = t.procedure.use(requireUser).query(async ({ ctx }) => {
  // Handle case where workspace is not in context (initial load scenarios)
  if (!ctx.workspace) {
    if (!ctx.tenant?.id) {
      // During first-time login, tenant might not be available yet
      // Throw error to indicate user needs workspace setup
      console.debug("No tenant ID available in context - user may be in auth setup phase");
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "No organization found - workspace setup required",
      });
    }

    // Try to fetch workspace directly from database using tenant/orgId
    try {
      let workspace = await db.query.workspaces.findFirst({
        where: (table, { eq, and, isNull }) =>
          and(eq(table.orgId, ctx.tenant.id), isNull(table.deletedAtM)),
        with: {
          quotas: true,
        },
      });

      // If no workspace found, this might be a newly created session/workspace
      // Add a small retry delay for database consistency
      if (!workspace) {
        console.debug("Workspace not found on first attempt, retrying after delay:", {
          orgId: ctx.tenant.id,
        });

        // Quick retry for database consistency
        await new Promise((resolve) => setTimeout(resolve, 50));

        workspace = await db.query.workspaces.findFirst({
          where: (table, { eq, and, isNull }) =>
            and(eq(table.orgId, ctx.tenant.id), isNull(table.deletedAtM)),
          with: {
            quotas: true,
          },
        });
      }

      if (!workspace) {
        console.debug("No workspace found for tenant after retry:", ctx.tenant.id);
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Workspace not found for organization - workspace setup required",
        });
      }

      const result = { ...workspace, quotas: workspace.quotas };

      const cacheKey = `workspace_${ctx.tenant?.id}`;
      workspaceCache.set(cacheKey, {
        data: result,
        timestamp: Date.now(),
      });

      return result;
    } catch (error) {
      // If it's already a TRPCError, re-throw it
      if (error instanceof TRPCError) {
        throw error;
      }

      console.warn("Failed to fetch workspace directly:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch workspace data",
        cause: error,
      });
    }
  }
  const cacheKey = `workspace_${ctx.tenant?.id}`;

  try {
    const cached = workspaceCache.get(cacheKey);
    const now = Date.now();

    // Return cached data if still valid
    if (cached && now - cached.timestamp < CACHE_TTL) {
      return cached.data;
    }

    // The workspace is already available in context from requireWorkspace middleware
    // but we need to fetch it with quotas and related data
    let workspace = await db.query.workspaces.findFirst({
      where: (table, { eq, and, isNull }) =>
        and(eq(table.orgId, ctx.tenant.id), isNull(table.deletedAtM)),
      with: {
        quotas: true,
      },
    });

    // Add retry logic for recently created workspaces
    if (!workspace) {
      console.debug("Workspace not found in context query, retrying:", {
        orgId: ctx.tenant.id,
      });

      // Quick retry for database consistency
      await new Promise((resolve) => setTimeout(resolve, 50));

      workspace = await db.query.workspaces.findFirst({
        where: (table, { eq, and, isNull }) =>
          and(eq(table.orgId, ctx.tenant.id), isNull(table.deletedAtM)),
        with: {
          quotas: true,
        },
      });
    }

    if (!workspace) {
      // Log for debugging but don't throw immediately
      console.debug("Workspace not found for context tenant after retry:", ctx.tenant.id);
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Workspace not found",
      });
    }

    const result = { ...workspace, quotas: workspace.quotas };

    // Cache the result
    workspaceCache.set(cacheKey, {
      data: result,
      timestamp: Date.now(),
    });

    return result;
  } catch (error) {
    // If it's already a TRPCError, re-throw it
    if (error instanceof TRPCError) {
      throw error;
    }

    // For database errors, provide more context
    console.warn("Database error fetching workspace:", error);
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message: "Failed to fetch workspace data",
      cause: error,
    });
  }
});

// Helper to clear cache when workspace changes
export const clearWorkspaceCache = (orgId: string) => {
  workspaceCache.delete(`workspace_${orgId}`);
};

// Helper to clear all workspace caches (useful for testing or memory management)
export const clearAllWorkspaceCaches = () => {
  workspaceCache.clear();
};

// Cleanup expired entries periodically (run this on a schedule if needed)
export const cleanupExpiredWorkspaceCaches = () => {
  const now = Date.now();
  let cleanedCount = 0;

  for (const [key, { timestamp }] of workspaceCache.entries()) {
    if (now - timestamp >= CACHE_TTL) {
      workspaceCache.delete(key);
      cleanedCount++;
    }
  }

  return cleanedCount;
};
