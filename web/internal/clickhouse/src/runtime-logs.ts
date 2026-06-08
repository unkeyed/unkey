import { Err, Ok, type Result } from "@unkey/error";
import { z } from "zod";
import type { QueryError } from "./client/error";
import type { Querier } from "./client/interface";

const TABLE = "default.runtime_logs_raw_v1";

const MAX_PAGE_SIZE = 1_000;

// Table is PARTITION BY toDate(inserted_at); grace covers ingestion lag so
// partition pruning still hits the right partitions for events near the edge.
const INGESTION_LAG_GRACE_MS = 2 * 60 * 60 * 1000;

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

// Read attributes_text (plain String) instead of the dynamic `attributes`
// JSON column: JSON fans out into ~1k subcolumn files per part, so on CH
// Cloud cold queries pay ~1k S3 GetObjects for the marks alone.
const runtimeLogRow = z.object({
  time: z.int(),
  severity: z.string(),
  message: z.string(),
  deployment_id: z.string(),
  region: z.string(),
  k8s_pod_name: z.string(),
  attributes_text: z.string().nullable(),
});

export function getRuntimeLogs(ch: Querier) {
  return async (args: RuntimeLogsRequest) => {
    const wheres: string[] = [
      "workspace_id = {workspaceId: String}",
      "project_id = {projectId: String}",
      "time BETWEEN {startTime: Int64} AND {endTime: Int64}",
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
      // lower() on both sides keeps the ngrambf_v1 skip index eligible.
      wheres.push("positionCaseInsensitive(lower(message), lower({message: String})) > 0");
    }
    if (args.k8sPodNames.length > 0) {
      wheres.push("k8s_pod_name IN {k8sPodNames: Array(String)}");
    }
    if (args.cursorTime !== null) {
      wheres.push("time < {cursorTime: Int64}");
    }

    const filterConditions = wheres.join("\n          AND ");

    // +1 to derive hasMore without a count(*).
    const pageSize = Math.min(Math.max(args.limit, 1), MAX_PAGE_SIZE);
    const fetchLimit = pageSize + 1;

    const logsQuery = ch.query({
      query: `
        SELECT
          time, severity, message, deployment_id,
          region, k8s_pod_name, attributes_text
        FROM ${TABLE}
        WHERE ${filterConditions}
        ORDER BY time DESC, deployment_id DESC
        LIMIT {limit: Int}
        SETTINGS
          optimize_read_in_order = 1,
          optimize_move_to_prewhere = 1,
          max_execution_time = ${MAX_EXECUTION_TIME_SECONDS}`,
      params: runtimeLogsQueryParamsSchema,
      schema: runtimeLogRow,
    });

    const rowsPromise = logsQuery({
      ...args,
      limit: fetchLimit,
      partitionStartTime: args.startTime - INGESTION_LAG_GRACE_MS,
      partitionEndTime: args.endTime + INGESTION_LAG_GRACE_MS,
    });

    return {
      logsQuery: rowsPromise.then(
        (res): Result<RuntimeLog[], QueryError> =>
          res.err ? Err(res.err) : Ok(res.val.map(toRuntimeLog)),
      ),
    };
  };
}

function toRuntimeLog(row: z.infer<typeof runtimeLogRow>): RuntimeLog {
  const { attributes_text, ...rest } = row;
  return { ...rest, attributes: parseAttributes(attributes_text) };
}

function parseAttributes(text: string | null): RuntimeLog["attributes"] {
  if (text === null || text === "" || text === "{}") {
    return null;
  }
  try {
    const parsed = JSON.parse(text);
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
      return parsed as RuntimeLog["attributes"];
    }
  } catch {
    // Malformed JSON: drop attributes rather than fail the whole page.
  }
  return null;
}
