import { NoopTinybird, Tinybird } from "@chronark/zod-bird";
import { z } from "zod";

export class Analytics {
  public readonly client: Tinybird | NoopTinybird;

  constructor(token?: string) {
    this.client = token ? new Tinybird({ token }) : new NoopTinybird();
  }

  public get ingestKeyVerification() {
    return this.client.buildIngestEndpoint({
      datasource: "key_verifications__v2",
      event: z.object({
        workspaceId: z.string(),
        apiId: z.string(),
        keyId: z.string(),
        deniedReason: z.enum(["RATE_LIMITED", "USAGE_EXCEEDED", "FORBIDDEN"]).optional(),
        time: z.number(),
        ipAddress: z.string().default(""),
        userAgent: z.string().default(""),
        requestedResource: z.string().default(""),
        edgeRegion: z.string().default(""),
        region: z.string(),
        // deprecated, use deniedReason
        ratelimited: z.boolean().default(false),
        // deprecated, use deniedReason
        usageExceeded: z.boolean().default(false),
      }),
    });
  }
}
