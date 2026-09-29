import { and, db, eq, ne, schema } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { z } from "zod";
import { requireProject } from "./access";
import { projectInput, target } from "./schemas";

export const listAppBindings = workspaceProcedure
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
        id: schema.appBindings.id,
        appId: schema.appBindings.appId,
        environmentId: schema.appBindings.environmentId,
        targetAppId: schema.appBindings.resourceId,
        targetAppName: schema.apps.name,
        targetAppSlug: schema.apps.slug,
        name: schema.appBindings.name,
        targetType: schema.appBindings.selectionMode,
        targetEnvironmentId: schema.appBindings.targetEnvironmentId,
        targetDeploymentId: schema.appBindings.targetDeploymentId,
      })
      .from(schema.appBindings)
      .innerJoin(schema.apps, eq(schema.apps.id, schema.appBindings.resourceId))
      .where(
        and(
          eq(schema.appBindings.workspaceId, ctx.workspace.id),
          eq(schema.appBindings.projectId, input.projectId),
          eq(schema.appBindings.resourceType, "app"),
          ne(schema.appBindings.resourceId, schema.appBindings.appId),
          eq(schema.apps.workspaceId, ctx.workspace.id),
          eq(schema.apps.projectId, input.projectId),
          input.appId ? eq(schema.appBindings.appId, input.appId) : undefined,
          input.environmentId
            ? eq(schema.appBindings.environmentId, input.environmentId)
            : undefined,
          input.targetAppId ? eq(schema.appBindings.resourceId, input.targetAppId) : undefined,
        ),
      )
      .orderBy(schema.appBindings.name)
      .limit(500);
    return rows.map((row) => ({ ...row, ...target.parse(row) }));
  });
