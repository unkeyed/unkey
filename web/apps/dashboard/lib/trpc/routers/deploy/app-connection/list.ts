import { and, db, eq, ne, schema } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { z } from "zod";
import { requireProject } from "./access";
import { projectInput, target } from "./schemas";

export const listAppConnections = workspaceProcedure
  .input(
    projectInput.extend({
      appId: z.string().min(1).optional(),
      environmentId: z.string().min(1).optional(),
      targetAppId: z.string().min(1).optional(),
    }),
  )
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx, input }) => {
    await requireProject(ctx.workspace.id, input.projectId);
    const rows = await db
      .select({
        id: schema.appConnections.id,
        appId: schema.appConnections.appId,
        environmentId: schema.appConnections.environmentId,
        targetAppId: schema.appConnections.resourceId,
        targetAppName: schema.apps.name,
        targetAppSlug: schema.apps.slug,
        name: schema.appConnections.name,
        targetType: schema.connectionAppTargets.selectionMode,
        targetEnvironmentId: schema.connectionAppTargets.targetEnvironmentId,
        targetDeploymentId: schema.connectionAppTargets.targetDeploymentId,
      })
      .from(schema.appConnections)
      .leftJoin(
        schema.connectionAppTargets,
        eq(schema.connectionAppTargets.connectionId, schema.appConnections.id),
      )
      .innerJoin(schema.apps, eq(schema.apps.id, schema.appConnections.resourceId))
      .where(
        and(
          eq(schema.appConnections.workspaceId, ctx.workspace.id),
          eq(schema.appConnections.projectId, input.projectId),
          eq(schema.appConnections.resourceType, "app"),
          ne(schema.appConnections.resourceId, schema.appConnections.appId),
          eq(schema.apps.workspaceId, ctx.workspace.id),
          eq(schema.apps.projectId, input.projectId),
          input.appId ? eq(schema.appConnections.appId, input.appId) : undefined,
          input.environmentId
            ? eq(schema.appConnections.environmentId, input.environmentId)
            : undefined,
          input.targetAppId ? eq(schema.appConnections.resourceId, input.targetAppId) : undefined,
        ),
      )
      .orderBy(schema.appConnections.name)
      .limit(500);
    return rows.map((row) => ({ ...row, ...target.parse(row) }));
  });
