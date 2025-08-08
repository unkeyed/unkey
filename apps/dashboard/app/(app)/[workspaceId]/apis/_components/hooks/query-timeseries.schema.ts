import { z } from "zod";

export const verificationQueryTimeseriesPayload = z.object({
  startTime: z.number().int(),
  endTime: z.number().int(),
  since: z.string(),
  keyspaceId: z.string(),
});

export type VerificationQueryTimeseriesPayload = z.infer<typeof verificationQueryTimeseriesPayload>;
