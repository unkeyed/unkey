import { z } from "zod";
import type { Querier } from "./client/interface";

const TABLE = "default.runtime_logs_raw_v1";

export const runtimeLogsRequestSchema = z.object({
  workspaceId: z.string(),
  projectId: z.string(),
  deploymentId: z.string().nullable(),
  environmentId: z.string(),
  appId: z.string(),
  limit: z.int(),
  startTime: z.int(),
  endTime: z.int(),
  severity: z.array(z.string()).nullable(),
  region: z.array(z.string()).nullable(),
  message: z.string().nullable(),
  k8sPodNames: z.array(z.string()),
  cursorTime: z.int().nullable(),
});

export type RuntimeLogsRequest = z.infer<typeof runtimeLogsRequestSchema>;

export const runtimeLog = z.object({
  time: z.int(),
  severity: z.string(),
  message: z.string(),
  deployment_id: z.string(),
  region: z.string(),
  k8s_pod_name: z.string(),
  attributes: z.record(z.string(), z.unknown()).nullable(),
});

export type RuntimeLog = z.infer<typeof runtimeLog>;

export function getRuntimeLogs(ch: Querier) {
  return async (args: RuntimeLogsRequest) => {
    const wheres: string[] = [
      "workspace_id = {workspaceId: String}",
      "project_id = {projectId: String}",
      "environment_id = {environmentId: String}",
      "app_id = {appId: String}",
      "time BETWEEN {startTime: UInt64} AND {endTime: UInt64}",
    ];

    if (args.deploymentId !== null) {
      wheres.push("deployment_id = {deploymentId: String}");
    }
    if (args.severity !== null && args.severity.length > 0) {
      wheres.push("severity IN {severity: Array(String)}");
    }
    if (args.region !== null && args.region.length > 0) {
      wheres.push("region IN {region: Array(String)}");
    }
    if (args.message !== null && args.message !== "") {
      // lower() on both sides so the ngrambf_v1(3, ...) skip index on
      // lower(message) can be used by the optimizer.
      wheres.push("positionCaseInsensitive(lower(message), lower({message: String})) > 0");
    }
    if (args.k8sPodNames.length > 0) {
      wheres.push("k8s_pod_name IN {k8sPodNames: Array(String)}");
    }
    if (args.cursorTime !== null) {
      wheres.push("time < {cursorTime: UInt64}");
    }

    const filterConditions = wheres.join("\n          AND ");

    // Fetch limit+1 so we can compute hasMore without a separate count(*).
    const fetchLimit = args.limit + 1;

    const logsQuery = ch.query({
      query: `
        SELECT
          time, severity, message, deployment_id,
          region, k8s_pod_name, attributes
        FROM ${TABLE}
        WHERE ${filterConditions}
        ORDER BY time DESC
        LIMIT ${fetchLimit}`,
      params: runtimeLogsRequestSchema,
      schema: runtimeLog,
    });

    return {
      logsQuery: logsQuery(args),
    };
  };
}
