import { event } from "@unkey/events";
import { z } from "zod";

export const queuePayload = z.object({
  workspaceId: z.string(),
  webhookId: z.string(),
  payload: event,
});

export type QueuePayload = z.infer<typeof queuePayload>;
