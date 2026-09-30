import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireOrgAdmin, workspaceProcedure } from "../../trpc";
import { getLocalTeamProvider } from "./local-team-provider";

export const removeMembership = workspaceProcedure
  .use(requireOrgAdmin)
  .input(
    z.object({
      membershipId: z.string(),
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
      return await authProvider.removeMembership(input.membershipId, input.orgId);
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to remove membership",
        cause: error,
      });
    }
  });
