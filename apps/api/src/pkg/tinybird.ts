import { Tinybird as Client } from "@chronark/zod-bird";
import { z } from "zod";




export class Tinybird {
  private readonly tb: Client;

  constructor(opts: { token: string }) {
    this.tb = new Client(opts);

  }

  public get ingestKeyVerification() {
    return this.tb.buildIngestEndpoint({
      datasource: "key_verifications__v2",
      event: z.object({
        workspaceId: z.string(),
        apiId: z.string(),
        keyId: z.string(),
        denied: z.enum(["RATE_LIMITED", "USAGE_EXCEEDED"]).optional(),
        time: z.number(),
        ipAddress: z.string().optional(),
        userAgent: z.string().optional(),
        requestedResource: z.string().optional(),
      })
    })
  }

}
