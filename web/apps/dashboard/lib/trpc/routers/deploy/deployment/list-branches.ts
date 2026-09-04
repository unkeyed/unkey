import { and, db, desc, eq, isNotNull, ne, sql } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { deployments } from "@unkey/db/src/schema";
import { z } from "zod";

// Branch options for the deployments filter. Derived from the whole history,
// not from the loaded page, so a branch that last deployed months ago is still
// selectable. Most recently deployed branches come first.
export const listDeploymentBranches = workspaceProcedure
  .input(z.object({ projectId: z.string(), appId: z.string() }))
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx, input }): Promise<string[]> => {
    const latestDeployedAt = sql<number>`max(${deployments.createdAt})`;
    const rows = await db
      .select({ branch: deployments.gitBranch })
      .from(deployments)
      .where(
        and(
          eq(deployments.workspaceId, ctx.workspace.id),
          eq(deployments.projectId, input.projectId),
          eq(deployments.appId, input.appId),
          isNotNull(deployments.gitBranch),
          ne(deployments.gitBranch, ""),
        ),
      )
      .groupBy(deployments.gitBranch)
      .orderBy(desc(latestDeployedAt))
      .limit(500);

    return rows.flatMap((row) => (row.branch ? [row.branch] : []));
  });
