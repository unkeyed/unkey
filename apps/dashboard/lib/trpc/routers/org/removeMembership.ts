import { auth as authProvider } from "@/lib/auth/server";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireOrgAdmin, requireOrgId, requireUser, t } from "../../trpc";

export const removeMembership = t.procedure
  .use(requireUser)
  .use(requireOrgAdmin)
  .use(requireOrgId)
  .input(
    z.object({
      membershipId: z.string(),
    }),
  )
  .mutation(async ({ input }) => {
    try {
      return await authProvider.removeMembership(input.membershipId);
    } catch (error) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to remove membership",
        cause: error,
      });
    }
  });
