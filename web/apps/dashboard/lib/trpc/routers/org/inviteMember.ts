import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireOrgAdmin, workspaceProcedure } from "../../trpc";
import { getLocalTeamProvider } from "./local-team-provider";

export const inviteMember = workspaceProcedure
  .use(requireOrgAdmin)
  .input(
    z.object({
      email: z.string().email(),
      orgId: z.string(), // needed for the requireOrgAdmin middleware
      role: z.enum(["basic_member", "admin"]),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const authProvider = getLocalTeamProvider();
    try {
      if (input.orgId !== ctx.workspace?.orgId) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: "Invalid organization ID",
        });
      }
      if (!ctx.workspace.quotas?.team) {
        throw new TRPCError({
          code: "FORBIDDEN",
          message: "Upgrade to Pro or Business to invite team members.",
        });
      }
      return await authProvider.inviteMember({
        email: input.email,
        role: input.role,
        orgId: input.orgId,
        inviterUserId: ctx.user.id,
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
