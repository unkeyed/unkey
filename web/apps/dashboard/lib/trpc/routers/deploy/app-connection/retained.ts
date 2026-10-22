import { and, countDistinct, db, eq, isNull, schema } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { z } from "zod";
import { requireApp } from "./access";
import { projectInput } from "./schemas";

export const listRetainedAppConnections = workspaceProcedure
  .input(projectInput.extend({ appId: z.string().min(1), environmentId: z.string().min(1) }))
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx, input }) => {
    await requireApp(ctx.workspace.id, input.projectId, input.appId);
    return db
      .select({
        id: schema.deploymentConnections.connectionId,
        targetAppName: schema.apps.name,
        deployments: countDistinct(schema.deploymentConnections.deploymentId),
      })
      .from(schema.deploymentConnections)
      .innerJoin(schema.apps, eq(schema.apps.id, schema.deploymentConnections.resourceId))
      .innerJoin(
        schema.deployments,
        and(
          eq(schema.deployments.id, schema.deploymentConnections.deploymentId),
          eq(schema.deployments.workspaceId, ctx.workspace.id),
          eq(schema.deployments.appId, input.appId),
          eq(schema.deployments.environmentId, input.environmentId),
        ),
      )
      .leftJoin(
        schema.appConnections,
        eq(schema.appConnections.id, schema.deploymentConnections.connectionId),
      )
      .where(
        and(
          eq(schema.deploymentConnections.workspaceId, ctx.workspace.id),
          eq(schema.deploymentConnections.projectId, input.projectId),
          eq(schema.deploymentConnections.appId, input.appId),
          eq(schema.deploymentConnections.environmentId, input.environmentId),
          eq(schema.deploymentConnections.resourceType, "app"),
          eq(schema.apps.workspaceId, ctx.workspace.id),
          eq(schema.apps.projectId, input.projectId),
          isNull(schema.appConnections.id),
        ),
      )
      .groupBy(schema.deploymentConnections.connectionId, schema.apps.name)
      .orderBy(schema.deploymentConnections.connectionId)
      .limit(500);
  });
