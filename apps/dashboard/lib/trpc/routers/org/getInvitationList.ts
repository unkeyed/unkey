import { requireOrgAdmin, requireUser, t } from "../../trpc";
import { auth as authProvider } from "@/lib/auth/server";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const getInvitationList = t.procedure
  .use(requireUser)
  .use(requireOrgAdmin)
  .input(z.string())
  .query(async ({input: orgId}) => {
      try {
        return await authProvider.getInvitationList(orgId);
        } catch (error) {
          console.error("Error retrieving organization member list:", error);
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message: "Failed to fetch organization member list",
            cause: error
          });
        }
    });