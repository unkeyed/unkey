import {
  DEPLOYMENT_STATUSES,
  isDeploymentInFlight,
} from "@/lib/collections/deploy/deployment-status";
import { and, db, eq, inArray, schema, sql } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { z } from "zod";

const CALLER_STATUSES = DEPLOYMENT_STATUSES.filter(
  (status) => status === "ready" || isDeploymentInFlight(status),
);

const RESULT_LIMIT = 10;

export const listDeploymentConnectionPins = workspaceProcedure
  .input(z.object({ deploymentId: z.string().min(1), projectId: z.string().min(1) }))
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx, input }) => {
    const [target] = await db
      .select({ appId: schema.deployments.appId })
      .from(schema.deployments)
      .where(
        and(
          eq(schema.deployments.id, input.deploymentId),
          eq(schema.deployments.workspaceId, ctx.workspace.id),
          eq(schema.deployments.projectId, input.projectId),
        ),
      )
      .limit(1);

    if (!target) {
      return { pins: [], truncated: false };
    }

    const current = await db
      .select({
        connectionId: schema.appConnections.id,
        connectionName: schema.appConnections.name,
        callerAppId: schema.appConnections.appId,
        callerAppName: schema.apps.name,
      })
      .from(schema.appConnections)
      .innerJoin(
        schema.connectionAppTargets,
        eq(schema.connectionAppTargets.connectionId, schema.appConnections.id),
      )
      .innerJoin(
        schema.apps,
        and(
          eq(schema.apps.id, schema.appConnections.appId),
          eq(schema.apps.workspaceId, ctx.workspace.id),
          eq(schema.apps.projectId, input.projectId),
        ),
      )
      .where(
        and(
          eq(schema.appConnections.workspaceId, ctx.workspace.id),
          eq(schema.appConnections.projectId, input.projectId),
          eq(schema.appConnections.resourceType, "app"),
          eq(schema.appConnections.resourceId, target.appId),
          eq(schema.connectionAppTargets.selectionMode, "deployment"),
          eq(schema.connectionAppTargets.targetDeploymentId, input.deploymentId),
        ),
      )
      .orderBy(schema.appConnections.name)
      .limit(RESULT_LIMIT + 1);

    const snapshots = await db
      .select({
        connectionId: schema.deploymentConnections.connectionId,
        connectionName: schema.deploymentConnections.name,
        callerAppId: schema.deploymentConnections.appId,
        callerAppName: schema.apps.name,
        callerDeploymentId: schema.deployments.id,
        callerStatus: schema.deployments.status,
      })
      .from(schema.deploymentConnections)
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
        schema.environments,
        and(
          eq(schema.environments.id, schema.deploymentConnections.environmentId),
          eq(schema.environments.appId, schema.deploymentConnections.appId),
          eq(schema.environments.workspaceId, ctx.workspace.id),
        ),
      )
      .innerJoin(
        schema.apps,
        and(
          eq(schema.apps.id, schema.deployments.appId),
          eq(schema.apps.workspaceId, ctx.workspace.id),
          eq(schema.apps.projectId, input.projectId),
        ),
      )
      .where(
        and(
          eq(schema.deploymentConnections.workspaceId, ctx.workspace.id),
          eq(schema.deploymentConnections.projectId, input.projectId),
          eq(schema.deploymentConnections.resourceType, "app"),
          eq(schema.deploymentConnections.resourceId, target.appId),
          eq(schema.deploymentConnectionAppTargets.selectionMode, "deployment"),
          eq(schema.deploymentConnectionAppTargets.targetDeploymentId, input.deploymentId),
          eq(schema.deployments.desiredState, "running"),
          inArray(schema.deployments.status, CALLER_STATUSES),
          sql`JSON_CONTAINS(${schema.deployments.capabilities}, 'true', '$.private_networking')`,
        ),
      )
      .orderBy(schema.deploymentConnections.name, schema.deployments.createdAt)
      .limit(RESULT_LIMIT + 1);

    const rows = [
      ...current.map((pin) => ({
        ...pin,
        provenance: "current_rule" as const,
      })),
      ...snapshots.map((pin) => ({
        ...pin,
        provenance: "deployment_snapshot" as const,
      })),
    ];

    return {
      pins: rows.slice(0, RESULT_LIMIT),
      truncated: rows.length > RESULT_LIMIT,
    };
  });
