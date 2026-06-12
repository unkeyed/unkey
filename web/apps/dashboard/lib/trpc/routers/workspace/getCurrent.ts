import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { protectedProcedure } from "../../trpc";

export const getCurrentWorkspace = protectedProcedure.query(async ({ ctx }) => {
  // createContext already resolved the workspace (with quotas) for this
  // request, so the common case costs no extra query.
  if (ctx.workspace) {
    return ctx.workspace;
  }

  if (!ctx.tenant?.id) {
    // The session has no organization yet (fresh sign-up before onboarding)
    throw new TRPCError({
      code: "NOT_FOUND",
      message: "No organization found - workspace setup required",
    });
  }

  // ctx.workspace is also unset when the context query failed (context
  // creation swallows database errors), so give the lookup one direct
  // attempt before reporting the workspace as missing.
  const orgId = ctx.tenant.id;
  let workspace: Awaited<
    ReturnType<typeof db.query.workspaces.findFirst<{ with: { quotas: true } }>>
  >;
  try {
    workspace = await db.query.workspaces.findFirst({
      where: (table, { eq, and, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
      with: {
        quotas: true,
      },
    });
  } catch (error) {
    console.warn("Database error fetching workspace:", error);
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message: "Failed to fetch workspace data",
      cause: error,
    });
  }

  if (!workspace) {
    throw new TRPCError({
      code: "NOT_FOUND",
      message: "Workspace not found for organization - workspace setup required",
    });
  }

  return workspace;
});
