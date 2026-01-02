import { z } from "zod";

export const keyOutcomesQueryPayload = z.object({
  startTime: z.number().int(),
  endTime: z.number().int(),
  keyId: z.string(),
  keyAuthId: z.string(),
});

export type KeyOutcomesQueryPayload = z.infer<typeof keyOutcomesQueryPayload>;
