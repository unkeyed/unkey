import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";

export const listEnvironments = workspaceProcedure
  .input(
    z.object({
      projectId: z.string(),
    }),
  )
  .query(async ({ ctx, input }) => {
    try {
      const rows = await db.query.environments.findMany({
        where: { workspaceId: ctx.workspace.id, projectId: input.projectId },
        columns: {
          id: true,
          projectId: true,
          slug: true,
        },
        with: {
          app: {
            columns: { id: true },
          },
        },
      });

      return rows.map((row) => ({
        id: row.id,
        projectId: row.projectId,
        slug: row.slug,
        appId: row.app?.id ?? "",
      }));
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
