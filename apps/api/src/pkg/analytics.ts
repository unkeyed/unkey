import { NoopTinybird, Tinybird as Client } from "@chronark/zod-bird";
import { z } from "zod";

export class Analytics {
  private readonly tb: Client | NoopTinybird;

  constructor(token?: string) {
    this.tb = token ? new Client({ token }) : new NoopTinybird();
  }

  public get ingestKeyVerification() {
    return this.tb.buildIngestEndpoint({
      datasource: "key_verifications__v2",
      event: z.object({
        workspaceId: z.string(),
        apiId: z.string(),
        keyId: z.string(),
        denied: z.enum(["RATE_LIMITED", "USAGE_EXCEEDED", "FORBIDDEN"]).optional(),
        time: z.number(),
        ipAddress: z.string().optional(),
        userAgent: z.string().optional(),
        requestedResource: z.string().optional(),
        edgeRegion: z.string().optional(),
        region: z.string().optional(),
      }),
    });
  }
}
