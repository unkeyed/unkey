import { auth as authProvider } from "@/lib/auth/server";
import { OrganizationScopeError } from "@/lib/auth/types";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireOrgAdmin, workspaceProcedure } from "../../trpc";

export const removeMembership = workspaceProcedure
  .use(requireOrgAdmin)
  .input(
    z.object({
      membershipId: z.string(),
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
      return await authProvider.removeMembership(input.membershipId, input.orgId);
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      if (error instanceof OrganizationScopeError) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Membership not found",
        });
      }
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to remove membership",
        cause: error,
      });
    }
  });
