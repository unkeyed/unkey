import { auth as authProvider } from "@/lib/auth/server";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireOrgAdmin, requireOrgId, requireUser, t } from "../../trpc";

export const updateMembership = t.procedure
  .use(requireUser)
  .use(requireOrgAdmin)
  .use(requireOrgId)
  .input(
    z.object({
      membershipId: z.string(),
      role: z.string(),
    }),
  )
  .mutation(async ({ input }) => {
    try {
      return await authProvider.updateMembership({
        membershipId: input.membershipId,
        role: input.role,
      });
    } catch (error) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to update membership",
        cause: error,
      });
    }
  });
