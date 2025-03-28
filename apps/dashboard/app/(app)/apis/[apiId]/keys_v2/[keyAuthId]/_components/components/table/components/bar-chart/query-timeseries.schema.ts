import { z } from "zod";

export const keysListQueryTimeseriesPayload = z.object({
  startTime: z.number().int(),
  endTime: z.number().int(),
  keyId: z.string(),
  keyAuthId: z.string(),
});

export type KeysListQueryTimeseriesPayload = z.infer<typeof keysListQueryTimeseriesPayload>;
