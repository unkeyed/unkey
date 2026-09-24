import { auth } from "@/lib/auth/server";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";

export const listAuditMembers = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx }) => {
    try {
      const members = await auth.getOrganizationMemberList(ctx.tenant.id);
      return members.data.map((m) => ({
        label: m.user.fullName ?? m.user.email,
        value: m.user.id,
      }));
    } catch (error) {
      console.error("Error retrieving organization members for audit filters:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch organization members",
        cause: error,
      });
    }
  });
