import { z } from "zod";
import type { Inserter } from "./client/interface";

export function insertSDKTelemetry(ch: Inserter) {
  return ch.insert({
    table: "telemetry.raw_sdks_v1",
    schema: z.object({
      request_id: z.string(),
      time: z.int(),
      runtime: z.string(),
      platform: z.string(),
      versions: z.array(z.string()),
    }),
  });
}
