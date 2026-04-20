import { z } from "zod";

export const DEFAULT_BUCKET_NAME = "unkey_mutations";

export const auditLogsQueryPayload = z.object({
  limit: z.number().int().positive().max(100).default(50),
  page: z.number().int().min(1).optional().default(1),
  startTime: z.number().int().optional(),
  endTime: z.number().int().optional(),
  since: z.string(),
  bucket: z.string().default(DEFAULT_BUCKET_NAME),
  events: z
    .object({
      filters: z.array(
        z.object({
          operator: z.literal("is"),
          value: z.string(),
        }),
      ),
    })
    .nullable(),
  users: z
    .object({
      filters: z.array(
        z.object({
          operator: z.literal("is"),
          value: z.string(),
        }),
      ),
    })
    .nullable(),
  rootKeys: z
    .object({
      filters: z.array(
        z.object({
          operator: z.literal("is"),
          value: z.string(),
        }),
      ),
    })
    .nullable(),
});

export type AuditLogsQueryPayload = z.infer<typeof auditLogsQueryPayload>;
