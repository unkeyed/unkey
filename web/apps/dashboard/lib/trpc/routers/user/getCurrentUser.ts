import { auth as authProvider } from "@/lib/auth/server";
import type { AuthenticatedUser } from "@/lib/auth/types";
import { TRPCError } from "@trpc/server";
import { protectedProcedure } from "../../trpc";

export const getCurrentUser = protectedProcedure.query(async ({ ctx }) => {
  try {
    // The sealed session cookie already carries the user's profile, so the
    // common case needs no provider API call. It is at most as stale as the
    // last session refresh.
    const user = ctx.user.profile ?? (await authProvider.getUser(ctx.user.id));
    if (!user) {
      throw new TRPCError({ code: "NOT_FOUND", message: "User not found" });
    }

    const result: AuthenticatedUser = {
      ...user,
      orgId: ctx.tenant?.id ?? null,
      role: ctx.tenant?.role ?? null,
    };
    return result;
  } catch (error) {
    if (error instanceof TRPCError) {
      throw error;
    }
    console.error("Error fetching current user:", error);
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message: "Failed to fetch user data",
      cause: error,
    });
  }
});
