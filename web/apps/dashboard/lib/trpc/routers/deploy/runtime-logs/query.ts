import { clickhouse } from "@/lib/clickhouse";
import { and, db, eq, inArray, schema } from "@/lib/db";
import {
  type RuntimeLogsResponseSchema,
  runtimeLogsRequestSchema,
  runtimeLogsResponseSchema,
} from "@/lib/schemas/runtime-logs.schema";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
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

    const environment = input.environmentId
      ? project.environments.find((e) => e.id === input.environmentId)
      : project.environments[0];

    if (!environment) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "No environment found for this project",
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
        return { logs: [], hasMore: false };
      }

      k8sPodNames = instances.map((inst) => inst.k8sName);
      for (const inst of instances) {
        knownK8sToInstanceId.set(toInstanceKey(inst.k8sName, inst.region.name), inst.id);
      }
    }

    const transformedInputs = transformFilters(input);
    const { logsQuery } = await clickhouse.runtimeLogs.logs({
      ...transformedInputs,
      k8sPodNames,
      workspaceId: ctx.workspace.id,
      projectId: project.id,
      deploymentId: input.deploymentId ?? null,
      environmentId: environment.id,
      appId: environment.appId,
    });

    const logsResult = await logsQuery;

    if (logsResult.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Something went wrong when fetching data from clickhouse.",
      });
    }

    // The CH query fetched limit+1 rows; if we got more than `limit`,
    // there is at least one more page.
    const allLogs = logsResult.val;
    const hasMore = allLogs.length > input.limit;
    const chLogs = hasMore ? allLogs.slice(0, input.limit) : allLogs;

    const unknownEntries = uniqueK8sRegionEntries(chLogs, knownK8sToInstanceId);
    const resolvedMapping = await resolveK8sNamesToInstanceIds(unknownEntries, ctx.workspace.id);
    const k8sNameToInstanceId = new Map([...knownK8sToInstanceId, ...resolvedMapping]);

    const logs = chLogs.map((log) => ({
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
      hasMore,
      nextCursor: logs.length > 0 ? logs[logs.length - 1].time : undefined,
    };

    return response;
  });
