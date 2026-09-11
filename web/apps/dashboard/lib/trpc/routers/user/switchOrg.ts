import { auth as authProvider } from "@/lib/auth/server";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { protectedProcedure } from "../../trpc";
import { getAvailableWorkspaces } from "../workspace/listAvailable";

export const switchOrg = protectedProcedure
  .input(z.string())
  .mutation(async ({ ctx, input: orgId }) => {
    try {
      const workspaces = await getAvailableWorkspaces(ctx.user.id, orgId);
      if (!workspaces.some((workspace) => workspace.orgId === orgId)) {
        throw new Error("Workspace unavailable");
      }
      const { newToken, expiresAt, session } = await authProvider.switchOrg(orgId);

      return {
        success: true,
        token: newToken,
        expiresAt,
        session,
      };
    } catch (error) {
      console.error("Error switching organization:", error);
      throw new TRPCError({
        code: "FORBIDDEN",
        message: "Unable to switch workspace",
      });
    }
  });
