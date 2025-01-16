import { z } from "zod";

export const queryLogsPayload = z.object({
  limit: z.number().int(),
  startTime: z.number().int(),
  endTime: z.number().int(),
  path: z.string().optional().nullable(),
  host: z.string().optional().nullable(),
  method: z.string().optional().nullable(),
  requestId: z.string().optional().nullable(),
  responseStatus: z.array(z.number().int()).nullable(),
  cursor: z
    .object({
      requestId: z.string().nullable(),
      time: z.number().nullable(),
    })
    .optional()
    .nullable(),
});
