import { auth as authProvider } from "@/lib/auth/server";
import { TRPCError } from "@trpc/server";
import { requireUser, t } from "../../trpc";

export const getOrganizationMemberList = t.procedure.use(requireUser).query(async ({ ctx }) => {
  try {
    return await authProvider.getOrganizationMemberList(ctx.tenant.id);
  } catch (error) {
    console.error("Error retrieving organization member list:", error);
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message: "Failed to fetch organization memberlist",
      cause: error,
    });
  }
});
