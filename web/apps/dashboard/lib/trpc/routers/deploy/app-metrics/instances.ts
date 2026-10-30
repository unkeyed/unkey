import { lastExitOf } from "@/lib/collections/deploy/last-exit";
import { and, db, eq, inArray, ne, schema } from "@/lib/db";
import type { InstanceRow } from "@/lib/metrics/app-metrics";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { currentDeploymentId, deployRole } from "./current-deployment";
import { deploySourceOf } from "./deploy-markers";
import {
  type EnvironmentScope,
  environmentScopeInput,
  loadEnvironmentScope,
  loadProductionRouting,
} from "./scope";

export const appMetricsInstances = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(environmentScopeInput)
  .query(async ({ ctx, input }): Promise<InstanceRow[]> => {
    const scope = await loadEnvironmentScope(ctx.workspace.id, input);
    const [instances, routing] = await Promise.all([
      db.query.instances.findMany({
        where: liveInstancesOf(scope),
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
      presence: { kind: "live", instanceId: instance.id, status: instance.status },
      region: { name: instance.region.name },
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

export function liveInstancesOf(scope: EnvironmentScope) {
  const d = schema.deployments;
  const i = schema.instances;
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
  return and(
    eq(i.workspaceId, scope.workspaceId),
    eq(i.appId, scope.appId),
    inArray(i.deploymentId, environmentDeployments),
    ne(i.status, "inactive"),
  );
}
