import { lastExitOf } from "@/lib/collections/deploy/last-exit";
import { and, db, eq, schema } from "@/lib/db";
import type { InstanceRow } from "@/lib/metrics/app-metrics";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { currentDeploymentId, deployRole } from "./current-deployment";
import { deploySourceOf } from "./deploy-markers";
import { environmentScopeInput, loadEnvironmentScope, loadProductionRouting } from "./scope";

export const appMetricsInstances = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(environmentScopeInput)
  .query(async ({ ctx, input }): Promise<InstanceRow[]> => {
    const scope = await loadEnvironmentScope(ctx.workspace.id, input);
    const d = schema.deployments;
    const environmentDeployments = db
      .select({ id: d.id })
      .from(d)
      .where(
        and(
          eq(d.workspaceId, scope.workspaceId),
          eq(d.appId, scope.appId),
          eq(d.environmentId, scope.environmentId),
        ),
      );

    const [instances, routing] = await Promise.all([
      db.query.instances.findMany({
        where: (table, { and, eq, inArray, ne }) =>
          and(
            eq(table.workspaceId, scope.workspaceId),
            eq(table.appId, scope.appId),
            inArray(table.deploymentId, environmentDeployments),
            ne(table.status, "inactive"),
          ),
        columns: {
          id: true,
          deploymentId: true,
          k8sName: true,
          cpuMillicores: true,
          memoryMib: true,
          storageMib: true,
          status: true,
          containerStatus: true,
        },
        with: {
          region: { columns: { name: true } },
          deployment: {
            columns: {
              status: true,
              createdAt: true,
              gitCommitSha: true,
              gitCommitMessage: true,
              gitCommitAuthorHandle: true,
              gitCommitAuthorAvatarUrl: true,
              imageRequested: true,
            },
          },
        },
        orderBy: (table, { asc }) => [asc(table.k8sName)],
      }),
      loadProductionRouting(scope),
    ]);

    const currentId = currentDeploymentId({
      routing,
      live: instances.map(({ deploymentId, deployment }) => ({
        id: deploymentId,
        status: deployment.status,
        createdAt: deployment.createdAt,
      })),
    });

    return instances.map((instance) => ({
      instanceId: instance.id,
      region: { name: instance.region.name },
      status: instance.status,
      role: deployRole(
        { id: instance.deploymentId, status: instance.deployment.status },
        currentId,
      ),
      lastExit: lastExitOf(instance.containerStatus),
      deployment: {
        id: instance.deploymentId,
        source: deploySourceOf(instance.deployment),
      },
      replica: {
        k8sName: instance.k8sName,
        cpuMillicores: instance.cpuMillicores,
        memoryMib: instance.memoryMib,
        storageMib: instance.storageMib,
      },
    }));
  });
