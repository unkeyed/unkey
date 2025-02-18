import { ClickHouse } from "@unkey/clickhouse";
import { env } from "./env";

export const clickhouse = new ClickHouse({ url: env().CLICKHOUSE_URL });
