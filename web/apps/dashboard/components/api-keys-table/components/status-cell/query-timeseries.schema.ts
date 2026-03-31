import { z } from "zod";

export const keyOutcomesQueryPayload = z.object({
  startTime: z.int(),
  endTime: z.int(),
  keyId: z.string(),
  keyAuthId: z.string(),
});

export type KeyOutcomesQueryPayload = z.infer<typeof keyOutcomesQueryPayload>;
