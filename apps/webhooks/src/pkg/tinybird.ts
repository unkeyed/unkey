import { Tinybird } from "@chronark/zod-bird";
import { z } from "zod";

export class Analytics {
  public readonly client: Tinybird;
  constructor(opts: { tinybirdToken: string }) {
    this.client = new Tinybird({ token: opts.tinybirdToken });
  }

  public get getVerificationsByOwnerId() {
    return this.client.buildPipe({
      pipe: "get_verifictions_by_keySpaceId__v1",
      parameters: z.object({
        workspaceId: z.string(),
        keySpaceId: z.string(),
        start: z.number(),
        end: z.number(),
      }),
      data: z.object({
        ownerId: z.string(),
        verifications: z.number(),
      }),
    });
  }
}
