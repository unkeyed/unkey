-- Give the frontline overhead column the public name gateway_latency.
--
-- The customer knows the product as the gateway. Frontline is the internal
-- service name. An ALIAS column adds the public name. It needs no rename, no
-- storage, and no backfill. ClickHouse expands the alias at read time for each
-- part. A writer continues to insert frontline_latency.
--
-- The analytics grant in pkg/clickhouse names gateway_latency. It does not name
-- frontline_latency. Thus a workspace user reads only the public name. A column
-- grant checks the name in the query, and not the expansion.

ALTER TABLE `default`.`frontline_requests_raw_v1`
  ADD COLUMN `gateway_latency` Int64 ALIAS `frontline_latency`;
