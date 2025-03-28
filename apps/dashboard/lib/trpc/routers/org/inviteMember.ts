import { requireOrgAdmin, requireUser, t } from "../../trpc";
import { auth as authProvider } from "@/lib/auth/server";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const inviteMember = t.procedure
  .use(requireUser)
  .use(requireOrgAdmin)
  .input(
    z.object({
      email: z.string(),
      orgId: z.string(), // needed for the requireOrgAdmin middleware
      role: z.enum(["basic_member", "admin"]),
    }),
  )
  .mutation(async ({ input }) => {
    try {
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
