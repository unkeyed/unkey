import { z } from "zod";
import type { Querier } from "./client";

const STEPS_TABLE = "default.build_steps_v1";

// ─────────────────────────────────────────────────────────────
// Request Schemas
// ─────────────────────────────────────────────────────────────

export const buildStepsRequestSchema = z.object({
  workspaceId: z.string(),
  projectId: z.string(),
  deploymentId: z.string(),
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
});

// ─────────────────────────────────────────────────────────────
// Type Exports
// ─────────────────────────────────────────────────────────────

export type BuildStepsRequest = z.infer<typeof buildStepsRequestSchema>;
export type BuildStep = z.infer<typeof buildStepSchema>;

// ─────────────────────────────────────────────────────────────
// Query Functions
// ─────────────────────────────────────────────────────────────

export function getBuildSteps(ch: Querier) {
  return async (args: BuildStepsRequest) => {
    const query = ch.query({
      query: `
        SELECT
          step_id, started_at, completed_at, name,
          cached, error
        FROM (
          SELECT
            step_id, started_at, completed_at, name,
            cached, error
          FROM ${STEPS_TABLE}
          WHERE workspace_id = {workspaceId: String}
            AND project_id = {projectId: String}
            AND deployment_id = {deploymentId: String}
          ORDER BY started_at DESC, completed_at DESC
          LIMIT 1 BY step_id
        )
        ORDER BY started_at ASC, step_id ASC`,
      params: buildStepsRequestSchema,
      schema: buildStepSchema,
    });
    return query(args);
  };
}
