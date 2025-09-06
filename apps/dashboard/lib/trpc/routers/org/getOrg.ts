import { auth as authProvider } from "@/lib/auth/server";
import { TRPCError } from "@trpc/server";
import { requireOrgId, requireUser, t } from "../../trpc";

export const getOrg = t.procedure
  .use(requireUser)
  .use(requireOrgId)
  .query(async ({ ctx }) => {
    try {
      return await authProvider.getOrg(ctx.orgId);
    } catch (error) {
      console.error("Error retrieving org information:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch organization",
        cause: error,
      });
    }
  });
