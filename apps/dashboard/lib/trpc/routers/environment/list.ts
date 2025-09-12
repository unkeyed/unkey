import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { requireUser, requireWorkspace, t } from "../../trpc";

export const listEnvironments = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .query(async ({ ctx }) => {
    try {
      return await db.query.environments.findMany({
        where: (table, { eq }) => eq(table.workspaceId, ctx.workspace.id),
        columns: {
          id: true,
          projectId: true,
          slug: true,
        },
      });
    } catch (_error) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch deployments",
      });
    }
  });
