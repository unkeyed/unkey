import { loadWorkspace } from "@/lib/db/load-workspace";
import { TRPCError } from "@trpc/server";
import { protectedProcedure } from "../../trpc";

export const getCurrentWorkspace = protectedProcedure.query(async ({ ctx }) => {
  if (ctx.workspace) {
    return ctx.workspace;
  }

  if (!ctx.tenant?.id) {
    // The session has no organization yet (fresh sign-up before onboarding).
    throw new TRPCError({
      code: "NOT_FOUND",
      message: "No organization found - workspace setup required",
    });
  }

  // Context creation swallows database errors, so retry before reporting a missing workspace.
  let workspace: Awaited<ReturnType<typeof loadWorkspace>>;
  try {
    workspace = await loadWorkspace(ctx.tenant.id);
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
