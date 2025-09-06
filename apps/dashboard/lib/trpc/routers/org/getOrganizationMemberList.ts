import { auth as authProvider } from "@/lib/auth/server";
import { TRPCError } from "@trpc/server";
import { requireOrgId, requireUser, t } from "../../trpc";

export const getOrganizationMemberList = t.procedure
  .use(requireUser)
  .use(requireOrgId)
  .query(async ({ ctx }) => {
    try {
      return await authProvider.getOrganizationMemberList(ctx.orgId);
    } catch (error) {
      console.error("Error retrieving organization member list:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch organization member list",
        cause: error,
      });
    }
  });
