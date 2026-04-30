import { z } from "zod";

// Provider-discriminated config schema. Each provider validates its own
// fields so the dashboard form can derive its UI from the same source the
// server uses for input validation.
export const axiomConfigSchema = z.object({
  dataset: z.string().trim().min(1),
  endpoint: z.string().trim().url().optional().or(z.literal("")),
});

export const drainConfigSchema = z.discriminatedUnion("provider", [
  z.object({ provider: z.literal("axiom"), config: axiomConfigSchema }),
]);

export const sourceSchema = z.enum(["runtime", "request"]);
export const environmentSchema = z.enum(["production", "preview"]);

// Filter shapes mirror the JSON column on log_drains. The optional
// include_bodies on the request half is the load-bearing privacy default:
// false unless the customer explicitly opts in.
export const runtimeFilterSchema = z.object({
  minSeverity: z.enum(["debug", "info", "warn", "error"]).optional(),
});

export const requestFilterSchema = z.object({
  statusCodes: z.array(z.string()).optional(),
  excludePaths: z.array(z.string()).optional(),
  includeBodies: z.boolean().optional(),
});

export const filtersSchema = z
  .object({
    runtime: runtimeFilterSchema.optional(),
    request: requestFilterSchema.optional(),
  })
  .default({});
