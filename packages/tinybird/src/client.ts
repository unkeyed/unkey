import type { Tinybird } from "@chronark/zod-bird";
import { z } from "zod";

export const publishKeyVerification = (tb: Tinybird) =>
  tb.buildIngestEndpoint({
    datasource: "key_verifications__v1",
    event: z.object({
      keyId: z.string(),
      apiId: z.string(),
      tenantId: z.string(),
      time: z.number(),
      ratelimited: z.boolean(),
    }),
  });
