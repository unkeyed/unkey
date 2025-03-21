import { requireUser, t } from "../../trpc";
import { auth as authProvider } from "@/lib/auth/server";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const getOrg = t.procedure
  .use(requireUser)
  .input(z.string())
  .query(async ({input: orgId}) => {
      try {
          return await authProvider.getOrg(orgId);
        } catch (error) {
          console.error("Error retrieving org information:", error);
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message: "Failed to fetch organization",
            cause: error
          });
        }
    });