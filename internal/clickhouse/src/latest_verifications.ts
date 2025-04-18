import { z } from "zod";
import type { Querier } from "./client";

const params = z.object({
  workspaceId: z.string(),
  keySpaceId: z.string(),
  keyId: z.string(),
  limit: z.number().int().positive().default(50).nullish(),
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
    FROM verifications.raw_key_verifications_v1
    WHERE workspace_id = {workspaceId: String}
    AND key_space_id = {keySpaceId: String}
    AND key_id = {keyId: String}
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
