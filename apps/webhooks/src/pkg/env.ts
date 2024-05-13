import { z } from "zod";
import type { QueuePayload } from "./schema";
export const zEnv = z.object({
  // VERSION: z.string().default("unknown"),
  DATABASE_HOST: z.string(),
  DATABASE_USERNAME: z.string(),
  DATABASE_PASSWORD: z.string(),
  UNKEY_WEBHOOK_KEYS_API_ID: z.string(),
  TINYBIRD_TOKEN: z.string(),
  WEBHOOKS_OUT: z.custom<Queue<QueuePayload>>((q) => typeof q === "object"),
  ENCRYPTION_KEYS: z.string().transform((s) =>
    z
      .array(
        z.object({
          version: z.number().int().min(1),
          key: z.string(),
        }),
      )
      .min(1)
      .parse(JSON.parse(s)),
  ),
});

export type Env = z.infer<typeof zEnv>;
