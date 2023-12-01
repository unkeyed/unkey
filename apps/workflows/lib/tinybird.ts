import { Tinybird as Client } from "@chronark/zod-bird";
import { z } from "zod";

const datetimeToUnixMilli = z.string().transform((t) => new Date(t).getTime());

export class Tinybird {
  private readonly tb: Client;

  constructor(token: string) {
    this.tb = new Client({ token });
  }

  public get activeKeys() {
    return this.tb.buildPipe({
      pipe: "endpoint_billing_get_active_keys_per_workspace_per_hour__v1",

      data: z.object({
        usage: z.number(),
        workspaceId: z.string(),
        time: datetimeToUnixMilli,
      }),
      opts: {
        cache: "no-store",
      },
    });
  }
  public get verifications() {
    return this.tb.buildPipe({
      pipe: "endpoint__billing_verifications_per_hour__v1",

      data: z.object({
        verifications: z.number(),
        workspaceId: z.string(),
        time: datetimeToUnixMilli,
      }),
      opts: {
        cache: "no-store",
      },
    });
  }
}
