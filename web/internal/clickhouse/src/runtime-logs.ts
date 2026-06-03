import { z } from "zod";
import type { Querier } from "./client/interface";

const TABLE = "default.runtime_logs_raw_v1";

const MAX_PAGE_SIZE = 1_000;

// The table is PARTITION BY toDate(inserted_at) (for clean TTL drops), so
// pruning partitions requires constraining inserted_at, not just `time`.
// This grace window covers ingestion lag between event time and arrival.
const INGESTION_LAG_GRACE_MS = 2 * 60 * 60 * 1000;

// Fail fast on the server instead of hanging until the HTTP client timeout.
const MAX_EXECUTION_TIME_SECONDS = 20;

export const runtimeLogsRequestSchema = z.object({
  workspaceId: z.string(),
  projectId: z.string(),
  deploymentId: z.string().nullable(),
  environmentId: z.array(z.string()),
  appId: z.string().nullable(),
  limit: z.int().min(1).max(MAX_PAGE_SIZE),
  startTime: z.int(),
  endTime: z.int(),
  severity: z.array(z.string()).nullable(),
  region: z.array(z.string()).nullable(),
  message: z.string().nullable(),
  k8sPodNames: z.array(z.string()),
  cursorTime: z.int().nullable(),
});

export type RuntimeLogsRequest = z.infer<typeof runtimeLogsRequestSchema>;

// Public shape + the n+1 limit bump for hasMore + derived inserted_at bounds.
const runtimeLogsQueryParamsSchema = runtimeLogsRequestSchema.extend({
  limit: z
    .int()
    .min(2)
    .max(MAX_PAGE_SIZE + 1),
  partitionStartTime: z.int(),
  partitionEndTime: z.int(),
});

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
      "app_id = {appId: String}",
      "time BETWEEN {startTime: Int64} AND {endTime: Int64}",
      // Prunes partitions; see INGESTION_LAG_GRACE_MS.
      `toDate(fromUnixTimestamp64Milli(inserted_at))
            BETWEEN toDate(fromUnixTimestamp64Milli({partitionStartTime: Int64}))
                AND toDate(fromUnixTimestamp64Milli({partitionEndTime: Int64}))`,
    ];

    // null appId = project-wide (every app); a value scopes to one app.
    if (args.appId !== null) {
      wheres.push("app_id = {appId: String}");
    }
    if (args.environmentId.length > 0) {
      wheres.push("environment_id IN {environmentId: Array(String)}");
    }
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
      // lower() on both sides so the ngrambf_v1 skip index on lower(message)
      // is eligible.
      wheres.push("positionCaseInsensitive(lower(message), lower({message: String})) > 0");
    }
    if (args.k8sPodNames.length > 0) {
      wheres.push("k8s_pod_name IN {k8sPodNames: Array(String)}");
    }
    if (args.cursorTime !== null) {
      wheres.push("time < {cursorTime: Int64}");
    }

    const filterConditions = wheres.join("\n          AND ");

    // +1 over the requested page size lets us compute hasMore without count(*).
    const pageSize = Math.min(Math.max(args.limit, 1), MAX_PAGE_SIZE);
    const fetchLimit = pageSize + 1;

    const logsQuery = ch.query({
      query: `
        SELECT
          time, severity, message, deployment_id,
          region, k8s_pod_name, attributes
        FROM ${TABLE}
        WHERE ${filterConditions}
        ORDER BY time DESC, deployment_id DESC
        LIMIT {limit: Int}
        SETTINGS
          optimize_read_in_order = 1,
          optimize_move_to_prewhere = 1,
          max_execution_time = ${MAX_EXECUTION_TIME_SECONDS}`,
      params: runtimeLogsQueryParamsSchema,
      schema: runtimeLog,
    });

    return {
      logsQuery: logsQuery({
        ...args,
        limit: fetchLimit,
        partitionStartTime: args.startTime - INGESTION_LAG_GRACE_MS,
        partitionEndTime: args.endTime + INGESTION_LAG_GRACE_MS,
      }),
    };
  };
}
