import { and, db, eq, schema } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { z } from "zod";
import { target } from "../app-connection/schemas";

const RESULT_LIMIT = 5;

export const listDeploymentConnections = workspaceProcedure
  .input(z.object({ deploymentId: z.string().min(1), projectId: z.string().min(1) }))
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx, input }) => {
    const rows = await db
      .select({
        id: schema.deploymentConnections.connectionId,
        targetAppName: schema.apps.name,
        targetType: schema.deploymentConnectionAppTargets.selectionMode,
        targetEnvironmentId: schema.deploymentConnectionAppTargets.targetEnvironmentId,
        targetEnvironmentSlug: schema.environments.slug,
        targetDeploymentId: schema.deploymentConnectionAppTargets.targetDeploymentId,
      })
      .from(schema.deploymentConnections)
      .innerJoin(
        schema.deployments,
        and(
          eq(schema.deployments.id, schema.deploymentConnections.deploymentId),
          eq(schema.deployments.workspaceId, schema.deploymentConnections.workspaceId),
          eq(schema.deployments.projectId, schema.deploymentConnections.projectId),
          eq(schema.deployments.appId, schema.deploymentConnections.appId),
          eq(schema.deployments.environmentId, schema.deploymentConnections.environmentId),
        ),
      )
      .innerJoin(
        schema.deploymentConnectionAppTargets,
        and(
          eq(
            schema.deploymentConnectionAppTargets.deploymentId,
            schema.deploymentConnections.deploymentId,
          ),
          eq(
            schema.deploymentConnectionAppTargets.connectionId,
            schema.deploymentConnections.connectionId,
          ),
        ),
      )
      .leftJoin(
        schema.apps,
        and(
          eq(schema.apps.id, schema.deploymentConnections.resourceId),
          eq(schema.apps.workspaceId, ctx.workspace.id),
          eq(schema.apps.projectId, input.projectId),
        ),
      )
      .leftJoin(
        schema.environments,
        and(
          eq(schema.environments.id, schema.deploymentConnectionAppTargets.targetEnvironmentId),
          eq(schema.environments.appId, schema.apps.id),
          eq(schema.environments.workspaceId, ctx.workspace.id),
          eq(schema.environments.projectId, input.projectId),
        ),
      )
      .where(
        and(
          eq(schema.deploymentConnections.deploymentId, input.deploymentId),
          eq(schema.deploymentConnections.workspaceId, ctx.workspace.id),
          eq(schema.deploymentConnections.projectId, input.projectId),
          eq(schema.deploymentConnections.resourceType, "app"),
        ),
      )
      .orderBy(schema.deploymentConnections.name, schema.deploymentConnections.connectionId)
      .limit(RESULT_LIMIT + 1);

    return {
      connections: rows.slice(0, RESULT_LIMIT).map((row) => ({ ...row, ...target.parse(row) })),
      truncated: rows.length > RESULT_LIMIT,
    };
  });
