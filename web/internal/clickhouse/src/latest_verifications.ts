import { z } from "zod";
import type { Querier } from "./client";

const params = z.object({
  workspaceId: z.string(),
  keySpaceId: z.string(),
  keyId: z.string(),
  limit: z.int().positive().prefault(50).nullish(),
});

export function getLatestVerifications(ch: Querier) {
  return async (args: z.infer<typeof params>) => {
    const query = ch.query({
      query: `
    SELECT
     time,
     outcome,
     region,
     tags
    FROM default.key_verifications_raw_v2
    PREWHERE workspace_id = {workspaceId: String}
    AND key_space_id = {keySpaceId: String}
    WHERE key_id = {keyId: String}
    ORDER BY time DESC
    LIMIT {limit: Int}`,
      params,
      schema: z.object({
        time: z.number(),
        outcome: z.string(),
        region: z.string(),
        tags: z.array(z.string()),
      }),
    });

    return query(args);
  };
}
