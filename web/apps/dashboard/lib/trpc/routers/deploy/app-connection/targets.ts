import { and, db, desc, eq, inArray, isNotNull, lt, ne, or, schema } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { z } from "zod";
import { requireApp } from "./access";
import { projectInput } from "./schemas";
import { mergeDeploymentTargets, pageTargetDeployments } from "./target-pagination";

const DEPLOYMENT_PAGE_SIZE = 100;

export const listAppConnectionTargets = workspaceProcedure
  .input(
    projectInput.extend({
      appId: z.string().min(1),
      environmentId: z.string().min(1),
      targetAppId: z.string().min(1).optional(),
      cursor: z.object({ createdAt: z.number().int(), id: z.string() }).nullish(),
    }),
  )
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx, input }) => {
    await requireApp(ctx.workspace.id, input.projectId, input.appId);
    if (input.targetAppId) {
      await requireApp(ctx.workspace.id, input.projectId, input.targetAppId);
    }
    const [apps, environments, pinnedConnections] = await Promise.all([
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
        .select({ deploymentId: schema.appConnections.targetDeploymentId })
        .from(schema.appConnections)
        .innerJoin(schema.apps, eq(schema.apps.id, schema.appConnections.resourceId))
        .where(
          and(
            eq(schema.appConnections.workspaceId, ctx.workspace.id),
            eq(schema.appConnections.projectId, input.projectId),
            eq(schema.appConnections.appId, input.appId),
            eq(schema.appConnections.environmentId, input.environmentId),
            eq(schema.appConnections.resourceType, "app"),
            ne(schema.appConnections.resourceId, schema.appConnections.appId),
            eq(schema.apps.workspaceId, ctx.workspace.id),
            eq(schema.apps.projectId, input.projectId),
            eq(schema.appConnections.selectionMode, "deployment"),
            isNotNull(schema.appConnections.targetDeploymentId),
          ),
        )
        .orderBy(schema.appConnections.name)
        .limit(500),
    ]);
    const deploymentSelect = {
      id: schema.deployments.id,
      appId: schema.deployments.appId,
      environmentId: schema.deployments.environmentId,
      status: schema.deployments.status,
      gitBranch: schema.deployments.gitBranch,
      image: schema.deployments.imageResolved,
      createdAt: schema.deployments.createdAt,
    };
    const pinnedIds = pinnedConnections.flatMap(({ deploymentId }) =>
      deploymentId ? [deploymentId] : [],
    );
    const [pinnedDeployments, choiceRows] = await Promise.all([
      pinnedIds.length > 0
        ? db
            .select(deploymentSelect)
            .from(schema.deployments)
            .where(
              and(
                eq(schema.deployments.workspaceId, ctx.workspace.id),
                eq(schema.deployments.projectId, input.projectId),
                inArray(schema.deployments.id, pinnedIds),
                isNotNull(schema.deployments.firstReadyAt),
              ),
            )
        : Promise.resolve([]),
      input.targetAppId
        ? db
            .select(deploymentSelect)
            .from(schema.deployments)
            .where(
              and(
                eq(schema.deployments.workspaceId, ctx.workspace.id),
                eq(schema.deployments.projectId, input.projectId),
                eq(schema.deployments.appId, input.targetAppId),
                isNotNull(schema.deployments.firstReadyAt),
                or(
                  eq(schema.deployments.status, "ready"),
                  eq(schema.deployments.status, "stopped"),
                ),
                input.cursor
                  ? or(
                      lt(schema.deployments.createdAt, input.cursor.createdAt),
                      and(
                        eq(schema.deployments.createdAt, input.cursor.createdAt),
                        lt(schema.deployments.id, input.cursor.id),
                      ),
                    )
                  : undefined,
              ),
            )
            .orderBy(desc(schema.deployments.createdAt), desc(schema.deployments.id))
            .limit(DEPLOYMENT_PAGE_SIZE + 1)
        : Promise.resolve([]),
    ]);
    const { page: choices, nextCursor } = pageTargetDeployments(choiceRows, DEPLOYMENT_PAGE_SIZE);
    const deployments = mergeDeploymentTargets(pinnedDeployments, choices).map(
      ({ createdAt: _createdAt, ...deployment }) => deployment,
    );
    return {
      apps,
      environments,
      deployments,
      nextCursor,
    };
  });
