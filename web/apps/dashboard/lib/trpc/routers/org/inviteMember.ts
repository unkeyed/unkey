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
      role: z.enum(["basic_member", "admin"]),
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
      return await authProvider.inviteMember({
        email: input.email,
        role: input.role,
        orgId: input.orgId,
      });
    } catch (error) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to invite member",
        cause: error,
      });
    }
  });
