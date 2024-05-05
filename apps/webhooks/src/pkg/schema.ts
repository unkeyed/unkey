import { webhookPayload } from "@unkey/webhooks";
import { z } from "zod";

export const queuePayload = z.object({
  payload: webhookPayload,
});

export type QueuePayload = z.infer<typeof queuePayload>;
