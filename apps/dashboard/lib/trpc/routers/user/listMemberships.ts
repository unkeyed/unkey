import { TRPCError } from "@trpc/server";
import { requireSelf, requireUser, t } from "../../trpc";
import { auth as authProvider } from "@/lib/auth/server";
import { z } from "zod";

export const listMemberships = t.procedure
  .use(requireUser)
  .use(requireSelf)
  .input(z.string())
  .query(async ({ input: userId }) => {
    try {
      return await authProvider.listMemberships(userId);
    } catch (error) {
      console.error("Error listing memberships:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch memberships",
        cause: error,
      });
    }
  });
