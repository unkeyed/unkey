import { createHmac } from "node:crypto";
import { auth as authProvider } from "@/lib/auth/server";
import { env } from "@/lib/env";
import { TRPCError } from "@trpc/server";
import { protectedProcedure } from "../../trpc";

export const getUserJotIdentity = protectedProcedure.query(async ({ ctx }) => {
  const { USERJOT_SECRET, NEXT_PUBLIC_USERJOT_PROJECT_ID } = env();
  if (!USERJOT_SECRET || !NEXT_PUBLIC_USERJOT_PROJECT_ID) {
    return null;
  }

  try {
    const user = await authProvider.getUser(ctx.user.id);
    if (!user) {
      throw new TRPCError({ code: "NOT_FOUND", message: "User not found" });
    }
    const signature = createHmac("sha256", USERJOT_SECRET).update(user.id).digest("hex");
    return {
      id: user.id,
      email: user.email,
      firstName: user.firstName,
      lastName: user.lastName,
      avatar: user.avatarUrl,
      signature,
    };
  } catch (error) {
    if (error instanceof TRPCError) {
      throw error;
    }
    console.error("Error building UserJot identity:", error);
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message: "Failed to build UserJot identity",
      cause: error,
    });
  }
});
