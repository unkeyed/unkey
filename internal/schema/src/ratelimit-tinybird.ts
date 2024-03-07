import { z } from "zod";

export const sharding = z.enum(["geo"]);

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
    sharding: sharding.optional(),
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
    userAgent: z.string().optional(),
    country: z.string().optional(),
    continent: z.string().optional(),
    city: z.string().optional(),
    colo: z.string().optional(),
  }),
});
