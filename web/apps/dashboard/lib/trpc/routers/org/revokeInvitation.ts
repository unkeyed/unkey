import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireOrgAdmin, workspaceProcedure } from "../../trpc";
import { getLocalTeamProvider } from "./local-team-provider";

export const revokeInvitation = workspaceProcedure
  .use(requireOrgAdmin)
  .input(
    z.object({
      invitationId: z.string(),
      orgId: z.string(), // needed for the requireOrgAdmin middleware
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const authProvider = getLocalTeamProvider();
    try {
      if (input.orgId !== ctx.workspace?.orgId) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: "Invalid organization ID",
        });
      }
      return await authProvider.revokeOrgInvitation(input.invitationId, input.orgId);
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to revoke invitation",
        cause: error,
      });
    }
  });
