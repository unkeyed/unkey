import { TRPCError } from "@trpc/server";
import { auth, t } from "../../trpc";
import { auth as authProvider } from "@/lib/auth/server";

export const listMemberships = t.procedure
  .use(auth)
  .query(async () => {
    try {
        return await authProvider.listMemberships();
      } catch (error) {
        console.error("Error listing memberships:", error);
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "Failed to fetch memberships",
          cause: error
        });
      }
  });