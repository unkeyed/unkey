import { z } from "zod";

export const checkResponseSchema = z.object({
  checks: z.array(
    z.object({
      status: z.number(),
      latency: z.number(),
    }),
  ),
});

export const checkRequestSchema = z.object({
  method: z.enum(["GET", "POST", "PUT", "PATCH", "DELETE"]),
  url: z.string().url(),
  n: z.number().int().positive().lte(10000).optional().default(1),
  headers: z.record(z.string()).optional(),
  body: z.string().optional(),
});
