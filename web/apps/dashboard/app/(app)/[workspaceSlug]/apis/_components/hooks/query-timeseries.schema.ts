import { z } from "zod";

export const verificationQueryTimeseriesPayload = z.object({
  startTime: z.int(),
  endTime: z.int(),
  since: z.string(),
  keyspaceId: z.string(),
});

export type VerificationQueryTimeseriesPayload = z.infer<typeof verificationQueryTimeseriesPayload>;
