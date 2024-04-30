import { Tinybird as Client } from "@chronark/zod-bird";
import { newId } from "@unkey/id";
import { z } from "zod";

export class Tinybird {
  private readonly tb: Client;

  constructor(token: string) {
    this.tb = new Client({ token });
  }

  public get activeKeys() {
    return this.tb.buildPipe({
      pipe: "endpoint__active_keys_by_workspace__v1",
      parameters: z.object({
        workspaceId: z.string(),
        year: z.number().int(),
        month: z.number().int().min(1).max(12),
      }),
      data: z.object({
        keys: z.number().int().nullable().default(0),
      }),
      opts: {
        cache: "no-store",
      },
    });
  }
  public get verifications() {
    return this.tb.buildPipe({
      pipe: "endpoint__verifications_by_workspace__v1",
      parameters: z.object({
        workspaceId: z.string(),
        year: z.number().int(),
        month: z.number().int().min(1).max(12),
      }),

      data: z.object({
        success: z.number().int().nullable().default(0),
        ratelimited: z.number().int().nullable().default(0),
        usageExceeded: z.number().int().nullable().default(0),
      }),
      opts: {
        cache: "no-store",
      },
    });
  }
  public get ratelimits() {
    return this.tb.buildPipe({
      pipe: "endpoint__ratelimits_by_workspace__v1",
      parameters: z.object({
        workspaceId: z.string(),
        year: z.number().int(),
        month: z.number().int().min(1).max(12),
      }),

      data: z.object({
        success: z.number().int().nullable().default(0),
        total: z.number().int().nullable().default(0),
      }),
      opts: {
        cache: "no-store",
      },
    });
  }
}
