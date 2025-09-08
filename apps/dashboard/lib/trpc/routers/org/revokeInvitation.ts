import { auth as authProvider } from "@/lib/auth/server";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireOrgAdmin, requireUser, t } from "../../trpc";

export const revokeInvitation = t.procedure
  .use(requireUser)
  .use(requireOrgAdmin)
  .input(
    z.object({
      invitationId: z.string(),
      orgId: z.string(), // needed for the requireOrgAdmin middleware
    }),
  )
  .mutation(async ({ ctx, input }) => {
    try {
      if (input.orgId !== ctx.workspace?.orgId) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: "Invalid organization ID",
        });
      }
      return await authProvider.revokeOrgInvitation(input.invitationId);
    } catch (error) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to revoke invitation",
        cause: error,
      });
    }
  });
