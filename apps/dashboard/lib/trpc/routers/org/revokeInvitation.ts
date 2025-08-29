import { auth as authProvider } from "@/lib/auth/server";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireOrgAdmin, requireOrgId, requireUser, t } from "../../trpc";

export const revokeInvitation = t.procedure
  .use(requireUser)
  .use(requireOrgAdmin)
  .use(requireOrgId)
  .input(
    z.object({
      invitationId: z.string(),
    }),
  )
  .mutation(async ({ input }) => {
    try {
      return await authProvider.revokeOrgInvitation(input.invitationId);
    } catch (error) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to revoke invitation",
        cause: error,
      });
    }
  });
