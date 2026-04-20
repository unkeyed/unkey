import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { frontlineRoutes, projects } from "@unkey/db/src/schema";
import { and, desc, eq } from "drizzle-orm";
import { z } from "zod";

export const listDomains = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(z.object({ projectId: z.string() }))
  .query(async ({ ctx, input }) => {
    // Query frontline_routes directly instead of using a relational query
    // through projects. The old approach used `findFirst` with
    // `with: { frontlineRoutes }` which generated a lateral join subquery.
    // When called per-project (N+1), each lateral join was expensive.
    // This flat query uses the project_id_idx index directly.
    const rows = await db
      .select({
        id: frontlineRoutes.id,
        projectId: frontlineRoutes.projectId,
        deploymentId: frontlineRoutes.deploymentId,
        environmentId: frontlineRoutes.environmentId,
        fullyQualifiedDomainName: frontlineRoutes.fullyQualifiedDomainName,
        sticky: frontlineRoutes.sticky,
        createdAt: frontlineRoutes.createdAt,
        updatedAt: frontlineRoutes.updatedAt,
      })
      .from(frontlineRoutes)
      .innerJoin(
        projects,
        and(eq(projects.id, frontlineRoutes.projectId), eq(projects.workspaceId, ctx.workspace.id)),
      )
      .where(eq(frontlineRoutes.projectId, input.projectId))
      .orderBy(desc(frontlineRoutes.updatedAt))
      .limit(500)
      .catch((error) => {
        console.error("Error querying domains:", error);
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "Failed to retrieve domains due to an error. If this issue persists, please contact support.",
        });
      });

    return rows;
  });
