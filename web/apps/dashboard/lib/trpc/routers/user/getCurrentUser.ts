import { auth as authProvider } from "@/lib/auth/server";
import type { AuthenticatedUser } from "@/lib/auth/types";
import { TRPCError } from "@trpc/server";
import { protectedProcedure } from "../../trpc";

export const getCurrentUser = protectedProcedure.query(async ({ ctx }) => {
  try {
    const user = await authProvider.getUser(ctx.user.id);
    return {
      ...user,
      orgId: ctx.tenant.id,
      role: ctx.tenant.role,
    } as AuthenticatedUser;
  } catch (error) {
    console.error("Error fetching current user:", error);
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message: "Failed to fetch user data",
      cause: error,
    });
  }
});
