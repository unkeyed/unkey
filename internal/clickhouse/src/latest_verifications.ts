import { z } from "zod";
import type { Querier } from "./client";

const params = z.object({
  workspaceId: z.string(),
  keySpaceId: z.string(),
  keyId: z.string(),
});
export function getLatestVerifications(ch: Querier) {
  return async (args: z.infer<typeof params>) => {
    const query = ch.query({
      query: `
    SELECT
     time,
     outcome,
     region,
    FROM default.raw_key_verifications_v1
    WHERE workspace_id = {workspaceId: String}
    AND key_space_id = {keySpaceId: String}
    AND key_id = {keyId: String}
    ORDER BY time DESC
    LIMIT 1`,
      params,
      schema: z.object({
        time: z.number(),
        outcome: z.string(),
        region: z.string(),
      }),
    });

    return query(args);
  };
}
