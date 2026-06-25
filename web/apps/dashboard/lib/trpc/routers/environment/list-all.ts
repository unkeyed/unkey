import { db, eq } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { environments } from "@unkey/db/src/schema";

export const listAllEnvironments = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx }) => {
    try {
      const rows = await db.query.environments.findMany({
        where: eq(environments.workspaceId, ctx.workspace.id),
        columns: {
          id: true,
          slug: true,
          projectId: true,
          appId: true,
        },
      });

      return rows.map((row) => ({
        id: row.id,
        name: row.slug,
        projectId: row.projectId,
        appId: row.appId,
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
