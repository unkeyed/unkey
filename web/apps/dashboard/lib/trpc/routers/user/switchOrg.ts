import { auth as authProvider } from "@/lib/auth/server";
import { z } from "zod";
import { protectedProcedure } from "../../trpc";

export const switchOrg = protectedProcedure.input(z.string()).mutation(async ({ input: orgId }) => {
  try {
    const { newToken, expiresAt, session } = await authProvider.switchOrg(orgId);

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
