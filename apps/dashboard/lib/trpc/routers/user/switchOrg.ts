import { auth as authProvider } from "@/lib/auth/server";
import { z } from "zod";
import { requireUser, t } from "../../trpc";

export const switchOrg = t.procedure
  .use(requireUser)
  .input(z.string())
  .mutation(async ({ input: orgId }) => {
    try {
      const { sessionToken, expiresAt } = await authProvider.switchOrg(orgId);
      return {
        success: true,
        token: sessionToken,
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
