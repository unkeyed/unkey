import { z } from "zod";

export const keysListQueryTimeseriesPayload = z.object({
  startTime: z.int(),
  endTime: z.int(),
  keyAuthId: z.string(),
  keyIds: z.array(z.string()),
});

export type KeysListQueryTimeseriesPayload = z.infer<typeof keysListQueryTimeseriesPayload>;
