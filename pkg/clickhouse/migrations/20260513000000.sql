-- Add inserted_at to sentinel_requests_raw_v1 so the logdrain coordinator
-- has a CH-side ingest timestamp to anchor its watermark on. Producer-set
-- `time` is corrupted by sentinel pod clock skew and any retransmits;
-- without an `inserted_at` column the cursor would silently drop rows
-- that land with a `time` value behind the cursor.
--
-- Non-breaking: the column has a server-side DEFAULT (toUnixTimestamp64Milli
-- of now64), so existing INSERTs that don't list the column still succeed
-- and existing SELECTs that don't reference it ignore it. Backfilled rows
-- pick up the now() value at the time the table receives the read of the
-- materialized default (which is "today" for every pre-migration row) —
-- that's fine because the coordinator only reads rows >= cursor and the
-- cursor is bootstrapped at first start to "now", not to the table's TTL
-- floor.
--
-- We do NOT reorder the sort key here. The cursor read on
-- sentinel_requests_raw_v1 will take a perf hit (the WHERE matches the
-- leading (workspace, project, env) prefix but the ORDER BY inserted_at
-- is not in the sort key, forcing CH to sort matching granules in-block).
-- Acceptable for v1 — drains are low-traffic at launch; promote to a
-- sentinel_requests_raw_v2 with the cursor-aligned sort key once any
-- single drain pushes more than ~1 MB/s.

ALTER TABLE default.sentinel_requests_raw_v1
    ADD COLUMN IF NOT EXISTS `inserted_at` Int64
        DEFAULT toUnixTimestamp64Milli(now64(3))
        CODEC(Delta(8), LZ4)
        AFTER `time`;
