import { requireOrgAdmin, requireUser, t } from "../../trpc";
import { auth as authProvider } from "@/lib/auth/server";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const updateMembership = t.procedure
  .use(requireUser)
  .use(requireOrgAdmin)
  .input(
    z.object({
      membershipId: z.string(),
      orgId: z.string(), // needed for the requireOrgAdmin middleware
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
