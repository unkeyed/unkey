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
  limit: z.number().int().positive().default(1000),
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

// ─────────────────────────────────────────────────────────────
// Type Exports
// ─────────────────────────────────────────────────────────────

export type BuildStepsRequest = z.infer<typeof buildStepsRequestSchema>;
export type BuildStepLogsRequest = z.infer<typeof buildStepLogsRequestSchema>;
export type BuildStep = z.infer<typeof buildStepSchema>;
export type BuildStepLog = z.infer<typeof buildStepLogSchema>;

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
