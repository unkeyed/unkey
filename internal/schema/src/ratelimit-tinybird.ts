import { z } from "zod";

export const sharding = z.enum(["edge", ""]);

export const ratelimitSchemaV1 = z.object({
  /**
   * The workspace owning this audit log
   */
  workspaceId: z.string(),
  namespaceId: z.string(),
  requestId: z.string(),
  identifier: z.string(),

  time: z.number(),
  serviceLatency: z.number(),
  success: z.boolean(),
  remaining: z.number().int(),
  config: z.object({
    limit: z.number().int(),
    duration: z.number().int(),
    async: z.boolean(),
    sharding: sharding.optional().default(""),
  }),
  resources: z.array(
    z.object({
      type: z.string(),
      id: z.string(),
      name: z.string().optional(),
      meta: z.record(z.union([z.string(), z.number(), z.boolean(), z.null()])).optional(),
    }),
  ),
  context: z.object({
    ipAddress: z.string(),
    userAgent: z.string().optional().default(""),
    country: z.string().optional().default(""),
    continent: z.string().optional().default(""),
    city: z.string().optional().default(""),
    colo: z.string().optional().default(""),
  }),
});
