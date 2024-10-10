import { env } from "@/lib/env";
import { type Clickhouse, Client, Noop } from "@unkey/clickhouse-zod";

const { CLICKHOUSE_URL } = env();

export const clickhouse: Clickhouse = CLICKHOUSE_URL
  ? new Client({ url: CLICKHOUSE_URL })
  : new Noop();
