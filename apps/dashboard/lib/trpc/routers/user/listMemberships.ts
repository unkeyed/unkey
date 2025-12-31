import { auth as authProvider } from "@/lib/auth/server";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { protectedProcedure, requireSelf } from "../../trpc";

export const listMemberships = protectedProcedure
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
