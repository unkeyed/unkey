import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireUser, requireWorkspace, t } from "../../trpc";

export const getWorkspace = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(
    z.object({
      orgId: z.string(),
    }),
  )
  .query(async ({ input }) => {
    const { orgId } = input;
    try {
      const workspace = await db.query.workspaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
        with: {
          apis: {
            where: (table, { isNull }) => isNull(table.deletedAtM),
          },
          quotas: true,
        },
      });
      return workspace;
    } catch (error) {
      console.error("Error fetching workspace:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch workspace",
        cause: error,
      });
    }
  });
