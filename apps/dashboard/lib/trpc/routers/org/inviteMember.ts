import { auth as authProvider } from "@/lib/auth/server";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireOrgAdmin, requireOrgId, requireUser, t } from "../../trpc";

export const inviteMember = t.procedure
  .use(requireUser)
  .use(requireOrgId)
  .use(requireOrgAdmin)
  .input(
    z.object({
      email: z.string(),
      role: z.enum(["basic_member", "admin"]),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    try {
      return await authProvider.inviteMember({
        email: input.email,
        role: input.role,
        orgId: ctx.orgId,
      });
    } catch (error) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to invite member",
        cause: error,
      });
    }
  });
