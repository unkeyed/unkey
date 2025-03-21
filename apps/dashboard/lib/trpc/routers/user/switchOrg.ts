import { requireUser, t } from "../../trpc";
import { auth as authProvider } from "@/lib/auth/server";
import { z } from "zod";

export const switchOrg = t.procedure
  .use(requireUser)
  .input(z.string())
  .mutation(async ({ input: orgId }) => {
    try {
      const { newToken, expiresAt } = await authProvider.switchOrg(orgId);
      return { 
        success: true, 
        token: newToken,
        expiresAt,
      };
    } catch (error) {
      console.error("Error switching organization:", error);
      return {
        success: false,
        error: error instanceof Error ? error.message : "Failed to switch organization",
      };
    }
  });