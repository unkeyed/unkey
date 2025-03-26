import { TRPCError } from "@trpc/server";
import { requireUser, t } from "../../trpc";
import { auth as authProvider } from "@/lib/auth/server";

export const getCurrentUser = t.procedure.use(requireUser).query(async () => {
  try {
    return await authProvider.getCurrentUser();
  } catch (error) {
    console.error("Error fetching current user:", error);
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message: "Failed to fetch user data",
      cause: error,
    });
  }
});
