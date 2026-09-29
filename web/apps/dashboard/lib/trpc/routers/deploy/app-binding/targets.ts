import { and, db, desc, eq, isNotNull, ne, schema } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { z } from "zod";
import { requireApp } from "./access";
import { projectInput } from "./schemas";

export const listAppBindingTargets = workspaceProcedure
  .input(projectInput.extend({ appId: z.string().min(1) }))
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx, input }) => {
    await requireApp(ctx.workspace.id, input.projectId, input.appId);
    const [apps, environments, deployments] = await Promise.all([
      db
        .select({
          id: schema.apps.id,
          name: schema.apps.name,
          slug: schema.apps.slug,
        })
        .from(schema.apps)
        .where(
          and(
            eq(schema.apps.workspaceId, ctx.workspace.id),
            eq(schema.apps.projectId, input.projectId),
            ne(schema.apps.id, input.appId),
          ),
        )
        .orderBy(schema.apps.name)
        .limit(100),
      db
        .select({
          id: schema.environments.id,
          appId: schema.environments.appId,
          slug: schema.environments.slug,
          kind: schema.environments.kind,
        })
        .from(schema.environments)
        .where(
          and(
            eq(schema.environments.workspaceId, ctx.workspace.id),
            eq(schema.environments.projectId, input.projectId),
          ),
        )
        .orderBy(schema.environments.slug)
        .limit(500),
      db
        .select({
          id: schema.deployments.id,
          appId: schema.deployments.appId,
          environmentId: schema.deployments.environmentId,
          status: schema.deployments.status,
          gitBranch: schema.deployments.gitBranch,
          image: schema.deployments.imageResolved,
        })
        .from(schema.deployments)
        .where(
          and(
            eq(schema.deployments.workspaceId, ctx.workspace.id),
            eq(schema.deployments.projectId, input.projectId),
            isNotNull(schema.deployments.firstReadyAt),
          ),
        )
        .orderBy(desc(schema.deployments.createdAt))
        .limit(100),
    ]);
    return {
      apps,
      environments,
      deployments: deployments.filter(
        (item) => item.status === "ready" || item.status === "stopped",
      ),
    };
  });
