import { auth as authProvider } from "@/lib/auth/server";
import { invalidateWorkspaceCache } from "@/lib/workspace-cache";
import { z } from "zod";
import { protectedProcedure } from "../../trpc";

export const switchOrg = protectedProcedure
  .input(z.string())
  .mutation(async ({ input: orgId, ctx }) => {
    try {
      const { newToken, expiresAt, session } = await authProvider.switchOrg(orgId);

      await invalidateWorkspaceCache(ctx.tenant.id); // Current org
      await invalidateWorkspaceCache(orgId); // Target org

      return {
        success: true,
        token: newToken,
        expiresAt,
        session,
      };
    } catch (error) {
      console.error("Error switching organization:", error);
      return {
        success: false,
        error: error instanceof Error ? error.message : "Failed to switch organization",
      };
    }
  });
