import { ORGANIZATION_ROLES } from "@/lib/auth/roles";
import { auth as authProvider } from "@/lib/auth/server";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireOrgAdmin, workspaceProcedure } from "../../trpc";

export const inviteMember = workspaceProcedure
  .use(requireOrgAdmin)
  .input(
    z.object({
      email: z.string(),
      orgId: z.string(), // needed for the requireOrgAdmin middleware
      role: z.enum(ORGANIZATION_ROLES),
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
      if (!ctx.workspace.limits?.teamEnabled) {
        throw new TRPCError({
          code: "FORBIDDEN",
          message: "Upgrade to Pro or Business to invite team members.",
        });
      }
      return await authProvider.inviteMember({
        email: input.email,
        role: input.role,
        orgId: input.orgId,
      });
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to invite member",
        cause: error,
      });
    }
  });
