import { auth as authProvider } from "@/lib/auth/server";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";

export const getOrg = workspaceProcedure.input(z.string()).query(async ({ ctx, input: orgId }) => {
  if (orgId !== ctx.workspace.orgId) {
    throw new TRPCError({
      code: "NOT_FOUND",
      message: "Organization not found",
    });
  }
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
