import { z } from "zod";

export const webhookPayload = z.discriminatedUnion("type", [
  z.object({
    type: z.literal("unkey.usage.record"),
    timestamp: z.string().datetime(),
    data: z.object({
      interval: z.object({
        start: z.number(),
        end: z.number(),
      }),
      keySpaceId: z.string(),
      records: z.array(
        z.object({
          ownerId: z.string(),
          verifications: z.number(),
        }),
      ),
    }),
  }),
]);

export const queuePayload = z.object({
  payload: webhookPayload,
});

export type QueuePayload = z.infer<typeof queuePayload>;
