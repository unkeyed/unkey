import { Tinybird as Client } from "@chronark/zod-bird";
import { z } from "zod";

export class Tinybird {
  private readonly tb: Client;

  constructor(token: string) {
    this.tb = new Client({ token });
  }

  public get activeKeys() {
    return this.tb.buildPipe({
      pipe: "get_billable_keys__v1",
      parameters: z.object({
        workspaceId: z.string(),
        /**
         * Unix milliseconds timestamp
         */
        start: z.number().int(),
        /**
         * Unix milliseconds timestamp
         */
        end: z.number().int(),
      }),

      data: z.object({
        activeKeys: z.number().int(),
      }),
      opts: {
        cache: "no-store",
      },
    });
  }

  public get verifications() {
    return this.tb.buildPipe({
      pipe: "get_billable_verifications__v1",
      parameters: z.object({
        workspaceId: z.string(),
        /**
         * Unix milliseconds timestamp
         */
        start: z.number().int(),
        /**
         * Unix milliseconds timestamp
         */
        end: z.number().int(),
      }),

      data: z.object({
        verifications: z.number().int(),
      }),
      opts: {
        cache: "no-store",
      },
    });
  }
}
