import { TRPCError } from "@trpc/server";
import { clickhouse } from "@/lib/clickhouse";
import { and, db, eq, inArray, schema } from "@/lib/db";
import {
  type RuntimeLogsResponseSchema,
  runtimeLogsRequestSchema,
  runtimeLogsResponseSchema,
} from "@/lib/schemas/runtime-logs.schema";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import {
  resolveK8sNamesToInstanceIds,
  toInstanceKey,
  transformFilters,
  uniqueK8sRegionEntries,
} from "./utils";

export const queryRuntimeLogs = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(runtimeLogsRequestSchema)
  .output(runtimeLogsResponseSchema)
  .query(async ({ ctx, input }) => {
    const project = await db.query.projects.findFirst({
      where: (table, { and, eq }) =>
        and(eq(table.id, input.projectId), eq(table.workspaceId, ctx.workspace.id)),
      columns: { id: true },
      with: {
        environments: {
          columns: { id: true, appId: true },
        },
      },
    });

    if (!project) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Project not found or access denied",
      });
    }

    // Resolve instanceIds to k8sPodNames for ClickHouse filtering,
    // and build the reverse map to avoid a redundant DB query later.
    const instanceIds = input.instanceId?.filters?.map((f) => f.value) ?? [];
    let k8sPodNames: string[] = [];
    const knownK8sToInstanceId = new Map<string, string>();

    if (instanceIds.length > 0) {
      const instances = await db.query.instances.findMany({
        where: and(
          inArray(schema.instances.id, instanceIds),
          eq(schema.instances.workspaceId, ctx.workspace.id),
        ),
        columns: { id: true, k8sName: true },
        with: { region: { columns: { name: true } } },
      });

      if (instances.length === 0) {
        return { logs: [], total: 0 };
      }

      k8sPodNames = instances.map((inst) => inst.k8sName);
      for (const inst of instances) {
        knownK8sToInstanceId.set(toInstanceKey(inst.k8sName, inst.region.name), inst.id);
      }
    }

    // If no app filter and no environment filter apply, the query reads all apps
    // and all environments of the project.
    const transformedInputs = transformFilters(input);

    // `environment_id` comes before `app_id` and `time` in the sort key. If only
    // `app_id` applies, ClickHouse cannot use `app_id` or the time bound to skip
    // granules. The environments of an app contain all rows of that app, so this
    // filter keeps the same rows. If the apps have no environments, the array
    // stays empty and no environment filter applies.
    if (input.appId.length > 0 && transformedInputs.environmentId.length === 0) {
      const selectedApps = new Set(input.appId);
      transformedInputs.environmentId = project.environments
        .filter((environment) => selectedApps.has(environment.appId))
        .map((environment) => environment.id);
    }

    const { logsQuery, totalQuery } = await clickhouse.runtimeLogs.logs(
      {
        ...transformedInputs,
        k8sPodNames,
        workspaceId: ctx.workspace.id,
        projectId: project.id,
        deploymentId: input.deploymentId,
        appId: input.appId,
      },
      { includeTotal: input.includeTotal },
    );

    const [logsResult, totalResult] = await Promise.all([logsQuery, totalQuery]);

    if (logsResult.err || totalResult.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Something went wrong when fetching data from clickhouse.",
      });
    }

    const chLogs = logsResult.val;
    const total = totalResult.val;

    const unknownEntries = uniqueK8sRegionEntries(chLogs, knownK8sToInstanceId);
    const resolvedMapping = await resolveK8sNamesToInstanceIds(unknownEntries, ctx.workspace.id);
    const k8sNameToInstanceId = new Map([...knownK8sToInstanceId, ...resolvedMapping]);

    const logs = chLogs.map((log) => ({
      log_id: log.log_id,
      time: log.time,
      severity: log.severity,
      message: log.message,
      deployment_id: log.deployment_id,
      region: log.region,
      instance_id: k8sNameToInstanceId.get(toInstanceKey(log.k8s_pod_name, log.region)) ?? "—",
      attributes: log.attributes,
    }));

    const response: RuntimeLogsResponseSchema = {
      logs,
      total,
    };

    return response;
  });
