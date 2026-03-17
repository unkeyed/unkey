import { clickhouse } from "@/lib/clickhouse";
import { db, eq, inArray, schema } from "@/lib/db";
import {
  type RuntimeLogsResponseSchema,
  runtimeLogsRequestSchema,
  runtimeLogsResponseSchema,
} from "@/lib/schemas/runtime-logs.schema";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { transformFilters } from "./utils";

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

    // Resolve instanceIds to k8sPodNames for ClickHouse filtering
    const instanceIds = input.instanceId?.filters?.map((f) => f.value) ?? [];
    let k8sPodNames: string[] = [];
    if (instanceIds.length > 0) {
      const instances = await db.query.instances.findMany({
        where: inArray(schema.instances.id, instanceIds),
        columns: { k8sName: true },
      });
      k8sPodNames = instances.map((inst) => inst.k8sName);
    }

    const transformedInputs = transformFilters(input);
    const { logsQuery, totalQuery } = await clickhouse.runtimeLogs.logs({
      ...transformedInputs,
      k8sPodNames,
      workspaceId: ctx.workspace.id,
      projectId: project.id,
      deploymentId: input.deploymentId ?? null,
      environmentId: environment.id,
      appId: environment.appId,
    });

    const [countResult, logsResult] = await Promise.all([totalQuery, logsQuery]);

    if (countResult.err || logsResult.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Something went wrong when fetching data from clickhouse.",
      });
    }

    const chLogs = logsResult.val;

    // Resolve k8s_pod_name to instance IDs for display
    const uniqueK8sNames = [...new Set(chLogs.map((log) => log.k8s_pod_name))];
    const k8sNameToInstanceId = new Map<string, string>();

    if (uniqueK8sNames.length > 0) {
      const instances = await db.query.instances.findMany({
        where: inArray(schema.instances.k8sName, uniqueK8sNames),
        columns: { id: true, k8sName: true },
      });
      for (const inst of instances) {
        k8sNameToInstanceId.set(inst.k8sName, inst.id);
      }
    }

    const logs = chLogs.map((log) => ({
      time: log.time,
      severity: log.severity,
      message: log.message,
      deployment_id: log.deployment_id,
      region: log.region,
      instance_id: k8sNameToInstanceId.get(log.k8s_pod_name) ?? log.k8s_pod_name,
      attributes: log.attributes,
    }));

    const response: RuntimeLogsResponseSchema = {
      logs,
      hasMore: logs.length === input.limit,
      total: countResult.val[0].total_count,
      nextCursor: logs.length > 0 ? logs[logs.length - 1].time : undefined,
    };

    return response;
  });
