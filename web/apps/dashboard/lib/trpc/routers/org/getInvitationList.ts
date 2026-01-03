import { auth as authProvider } from "@/lib/auth/server";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireOrgAdmin, workspaceProcedure } from "../../trpc";

export const getInvitationList = workspaceProcedure
  .use(requireOrgAdmin)
  .input(z.string())
  .query(async ({ ctx, input: orgId }) => {
    try {
      if (orgId !== ctx.workspace?.orgId) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: "Invalid organization ID",
        });
      }
      return await authProvider.getInvitationList(orgId);
    } catch (error) {
      console.error("Error retrieving organization member list:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch organization member list",
        cause: error,
      });
    }
  });
