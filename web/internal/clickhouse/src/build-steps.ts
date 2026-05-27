import { z } from "zod";
import type { Querier } from "./client";

const STEPS_TABLE = "default.build_steps_v1";
const LOGS_TABLE = "default.build_step_logs_v1";

// ─────────────────────────────────────────────────────────────
// Request Schemas
// ─────────────────────────────────────────────────────────────

export const buildStepsRequestSchema = z.object({
  workspaceId: z.string(),
  projectId: z.string(),
  deploymentId: z.string(),
});

export const buildStepLogsRequestSchema = buildStepsRequestSchema.extend({
  stepIds: z.array(z.string()),
  limit: z.number().int().positive().default(20),
});

export const buildStepsWithLogsRequestSchema = buildStepsRequestSchema.extend({
  logLimit: z.number().int().positive().default(20),
});

// ─────────────────────────────────────────────────────────────
// Response Schemas
// ─────────────────────────────────────────────────────────────

export const buildStepSchema = z.object({
  step_id: z.string(),
  started_at: z.number().int(),
  completed_at: z.number().int(),
  name: z.string(),
  cached: z.boolean(),
  error: z.string().transform((s) => (s === "" ? null : s)),
  has_logs: z.boolean(),
});

export const buildStepLogSchema = z.object({
  time: z.number().int(),
  step_id: z.string(),
  message: z.string(),
});

// Rows from getBuildStepsWithLogs. Logs are returned as two parallel arrays
// (log_times, log_messages) instead of a tuple array because ClickHouse JSON
// renders tuples as positional [t, m] pairs, which Zod can't validate as
// nicely as a parallel-array shape. Callers zip them back into
// {time, message} objects.
export const buildStepWithLogsRowSchema = buildStepSchema.extend({
  log_times: z.array(z.number().int()),
  log_messages: z.array(z.string()),
});

// ─────────────────────────────────────────────────────────────
// Type Exports
// ─────────────────────────────────────────────────────────────

export type BuildStepsRequest = z.infer<typeof buildStepsRequestSchema>;
export type BuildStepLogsRequest = z.infer<typeof buildStepLogsRequestSchema>;
export type BuildStepsWithLogsRequest = z.infer<typeof buildStepsWithLogsRequestSchema>;
export type BuildStep = z.infer<typeof buildStepSchema>;
export type BuildStepLog = z.infer<typeof buildStepLogSchema>;
export type BuildStepWithLogsRow = z.infer<typeof buildStepWithLogsRowSchema>;

// ─────────────────────────────────────────────────────────────
// Query Functions
// ─────────────────────────────────────────────────────────────

export function getBuildSteps(ch: Querier) {
  return async (args: BuildStepsRequest) => {
    const query = ch.query({
      query: `
        SELECT
          step_id, started_at, completed_at, name,
          cached, error, has_logs
        FROM ${STEPS_TABLE}
        WHERE workspace_id = {workspaceId: String}
          AND project_id = {projectId: String}
          AND deployment_id = {deploymentId: String}
        ORDER BY started_at ASC`,
      params: buildStepsRequestSchema,
      schema: buildStepSchema,
    });
    return query(args);
  };
}

export function getBuildStepLogs(ch: Querier) {
  return async (args: BuildStepLogsRequest) => {
    const query = ch.query({
      query: `
        SELECT time, step_id, message
        FROM ${LOGS_TABLE}
        WHERE workspace_id = {workspaceId: String}
          AND project_id = {projectId: String}
          AND deployment_id = {deploymentId: String}
          AND step_id IN {stepIds: Array(String)}
        ORDER BY time ASC, step_id ASC
        LIMIT {limit: Int}`,
      params: buildStepLogsRequestSchema,
      schema: buildStepLogSchema,
    });
    return query(args);
  };
}

// One round trip instead of two. The dashboard's build-steps handler called
// getBuildSteps + getBuildStepLogs back-to-back; on a managed CH (Cloud)
// that's one extra network RTT per refresh, and the dashboard polls every
// second.
//
// The LIMIT applies to the inner logs subquery (global cap across all
// steps), matching the previous handler's behavior. groupArray on the join
// result produces parallel arrays per step.
//
// `join_use_nulls = 1` is required: without it, ClickHouse's LEFT JOIN
// returns column defaults (0, '') for unmatched rows instead of NULL, and
// groupArray happily collects those defaults into [0]/[''] arrays. With
// the setting, unmatched right-side columns are NULL and groupArray skips
// them, producing the empty arrays we want for steps with no logs.
export function getBuildStepsWithLogs(ch: Querier) {
  return async (args: BuildStepsWithLogsRequest) => {
    const query = ch.query({
      query: `
        SELECT
          s.step_id      AS step_id,
          s.started_at   AS started_at,
          s.completed_at AS completed_at,
          s.name         AS name,
          s.cached       AS cached,
          s.error        AS error,
          s.has_logs     AS has_logs,
          groupArray(l.time)    AS log_times,
          groupArray(l.message) AS log_messages
        FROM ${STEPS_TABLE} AS s
        LEFT JOIN (
          SELECT step_id, time, message
          FROM ${LOGS_TABLE}
          WHERE workspace_id = {workspaceId: String}
            AND project_id = {projectId: String}
            AND deployment_id = {deploymentId: String}
          ORDER BY time ASC
          LIMIT {logLimit: UInt32}
        ) AS l ON l.step_id = s.step_id
        WHERE s.workspace_id = {workspaceId: String}
          AND s.project_id = {projectId: String}
          AND s.deployment_id = {deploymentId: String}
        GROUP BY s.step_id, s.started_at, s.completed_at, s.name, s.cached, s.error, s.has_logs
        ORDER BY s.started_at ASC
        SETTINGS join_use_nulls = 1`,
      params: buildStepsWithLogsRequestSchema,
      schema: buildStepWithLogsRowSchema,
    });
    return query(args);
  };
}
