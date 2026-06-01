import type { App } from "@/lib/collections/deploy/apps";
import { and, db, desc, eq } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { apps } from "@unkey/db/src/schema";
import { z } from "zod";

export const listApps = workspaceProcedure
  .input(z.object({ projectId: z.string() }))
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx, input }): Promise<App[]> => {
    const rows = await db
      .select({
        id: apps.id,
        projectId: apps.projectId,
        name: apps.name,
        slug: apps.slug,
        defaultBranch: apps.defaultBranch,
        currentDeploymentId: apps.currentDeploymentId,
        isRolledBack: apps.isRolledBack,
      })
      .from(apps)
      .where(and(eq(apps.workspaceId, ctx.workspace.id), eq(apps.projectId, input.projectId)))
      .orderBy(desc(apps.updatedAt), desc(apps.id));

    return rows.map((row) => ({
      id: row.id,
      projectId: row.projectId,
      name: row.name,
      slug: row.slug,
      defaultBranch: row.defaultBranch,
      currentDeploymentId: row.currentDeploymentId ?? null,
      isRolledBack: Boolean(row.isRolledBack),
    }));
  });
