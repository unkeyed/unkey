import { type Clickhouse, Client, Noop } from "@unkey/clickhouse-zod";
import { z } from "zod";
import { env } from "../env";

export async function getLastUsed(args: {
  workspaceId: string;
  keySpaceId: string;
  keyId: string;
}) {
  const { CLICKHOUSE_URL } = env();

  const ch: Clickhouse = CLICKHOUSE_URL ? new Client({ url: CLICKHOUSE_URL }) : new Noop();
  const query = ch.query({
    query: `
    SELECT
      time,
    FROM verifications.raw_key_verifications_v1
    WHERE 
      workspace_id = {workspaceId: String}
    AND key_space_id = {keySpaceId: String}
    AND key_id = {keyId:String}
    ORDER BY time DESC
    LIMIT 1
    ;`,
    params: z.object({
      workspaceId: z.string(),
      keySpaceId: z.string(),
      keyId: z.string(),
    }),
    schema: z.object({
      time: z.number().int(),
    }),
  });

  return query(args);
}
