import { z } from "zod";

export const DEFAULT_BUCKET_NAME = "unkey_mutations";
export const auditQueryLogsPayload = z.object({
  limit: z.number().int(),
  startTime: z.number().int(),
  endTime: z.number().int(),
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
  cursor: z
    .object({
      auditId: z.string().nullable(),
      time: z.number().nullable(),
    })
    .optional()
    .nullable(),
});

export type AuditQueryLogsPayload = z.infer<typeof auditQueryLogsPayload>;
