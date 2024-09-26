import { type Clickhouse, Client, Noop } from "@unkey/clickhouse-zod";
import { z } from "zod";
import { env } from "../env";

const params = z.object({
  workspaceId: z.string(),
  keySpaceId: z.string(),
  keyId: z.string(),
});
// dummy example of how to query stuff from clickhouse
export async function getLatestVerifications(args: z.infer<typeof params>) {
  const { CLICKHOUSE_URL } = env();

  const ch: Clickhouse = CLICKHOUSE_URL ? new Client({ url: CLICKHOUSE_URL }) : new Noop();
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
    LIMIT {limit: Int}`,
    params,
    schema: z.object({
      time: z.number(),
      outcome: z.string(),
      region: z.string(),
    }),
  });

  return query(args);
}
