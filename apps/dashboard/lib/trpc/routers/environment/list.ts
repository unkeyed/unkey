import { and, db, eq } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { environments } from "@unkey/db/src/schema";
import { z } from "zod";
import { requireUser, requireWorkspace, t } from "../../trpc";

export const listEnvironments = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(
    z.object({
      projectId: z.string(),
    }),
  )
  .query(async ({ ctx, input }) => {
    try {
      return await db.query.environments.findMany({
        where: and(
          eq(environments.workspaceId, ctx.workspace.id),
          eq(environments.projectId, input.projectId),
        ),
        columns: {
          id: true,
          projectId: true,
          slug: true,
        },
      });
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      console.error("Failed to fetch environments:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch environments",
      });
    }
  });
