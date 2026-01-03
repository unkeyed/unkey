import { auth as authProvider } from "@/lib/auth/server";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { protectedProcedure } from "../../trpc";

export const getOrg = protectedProcedure.input(z.string()).query(async ({ input: orgId }) => {
  try {
    return await authProvider.getOrg(orgId);
  } catch (error) {
    console.error("Error retrieving org information:", error);
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message: "Failed to fetch organization",
      cause: error,
    });
  }
});
