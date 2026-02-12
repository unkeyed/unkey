import { z } from "zod";
import type { Querier } from "./client/interface";

const TABLE = "default.runtime_logs_raw_v1";

export const runtimeLogsRequestSchema = z.object({
  workspaceId: z.string(),
  projectId: z.string(),
  deploymentId: z.string().nullable(),
  environmentId: z.string(),
  limit: z.int(),
  startTime: z.int(),
  endTime: z.int(),
  severity: z.array(z.string()).nullable(),
  message: z.string().nullable(),
  cursorTime: z.int().nullable(),
});

export type RuntimeLogsRequest = z.infer<typeof runtimeLogsRequestSchema>;

export const runtimeLog = z.object({
  time: z.int(),
  severity: z.string(),
  message: z.string(),
  deployment_id: z.string(),
  region: z.string(),
  attributes: z.record(z.string(), z.unknown()).nullable(),
});

export type RuntimeLog = z.infer<typeof runtimeLog>;

export function getRuntimeLogs(ch: Querier) {
  return async (args: RuntimeLogsRequest) => {
    const filterConditions = `
      workspace_id = {workspaceId: String}
      AND project_id = {projectId: String}
      AND (
        {deploymentId: Nullable(String)} IS NULL
        OR deployment_id = assumeNotNull({deploymentId: Nullable(String)})
      )
      AND environment_id = {environmentId: String}
      AND time BETWEEN {startTime: UInt64} AND {endTime: UInt64}

      AND (
        CASE
          WHEN length({severity: Array(String)}) > 0 THEN
            severity IN {severity: Array(String)}
          ELSE TRUE
        END
      )

    AND (
      {message: Nullable(String)} IS NULL
      OR {message: Nullable(String)} = ''
      OR positionCaseInsensitive(message, assumeNotNull({message: Nullable(String)})) > 0
    )
    `;

    const totalQuery = ch.query({
      query: `
        SELECT count(*) as total_count
        FROM ${TABLE}
        WHERE ${filterConditions}`,
      params: runtimeLogsRequestSchema,
      schema: z.object({ total_count: z.int() }),
    });

    const logsQuery = ch.query({
      query: `
        SELECT
          time, severity, message, deployment_id,
          region, attributes
        FROM ${TABLE}
        WHERE ${filterConditions}
          AND ({cursorTime: Nullable(UInt64)} IS NULL OR time < {cursorTime: Nullable(UInt64)})
        ORDER BY time DESC
        LIMIT {limit: Int}`,
      params: runtimeLogsRequestSchema,
      schema: runtimeLog,
    });

    return {
      logsQuery: logsQuery(args),
      totalQuery: totalQuery(args),
    };
  };
}
