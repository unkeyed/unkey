# Iceberg Migrations

This directory contains Iceberg table schema definitions that mirror the ClickHouse schema defined in `pkg/clickhouse/schema/`.

## Schema Mapping

These migrations create the same logical tables as ClickHouse but using Iceberg table format:

- `001_key_verifications_raw_v2.sql` → mirrors `pkg/clickhouse/schema/001_key_verifications_raw_v2.sql`
- `002_ratelimits_raw_v2.sql` → mirrors `pkg/clickhouse/schema/006_ratelimits_raw_v2.sql`
- `003_api_requests_raw_v2.sql` → mirrors `pkg/clickhouse/schema/012_api_requests_raw_v2.sql`

## Key Differences from ClickHouse

### Data Types
- `Int64` → `BIGINT`
- `Float64` → `DOUBLE`
- `Bool` → `BOOLEAN`
- `Array(String)` → `ARRAY<STRING>`
- `Map(String, Array(String))` → `MAP<STRING, ARRAY<STRING>>`
- `LowCardinality(String)` → `STRING` (Iceberg/Parquet handles cardinality automatically)

### Storage Features
- **Engine**: Uses `iceberg` format instead of ClickHouse's `MergeTree`
- **Compression**: Uses `zstd` compression via Parquet instead of ClickHouse codecs
- **Partitioning**: Partitioned by `days(time)` and `workspace_id` for efficient querying
- **TTL**: Replaced with Iceberg's snapshot retention (`history.expire.max-snapshot-age-ms`)
- **Indexes**: Not needed - Iceberg handles metadata indexing automatically
- **Deduplication**: Handled at application level instead of ClickHouse's `non_replicated_deduplication_window`

## Usage

These SQL files are used by the Iceberg writer implementation to create tables in customer-specific data lakes. Each workspace gets its own isolated bucket/namespace with these tables.

## Creating Tables

Tables are typically created automatically by the analytics system when first writing to a workspace's data lake. The creation process:

1. Check if table exists in the workspace's data lake
2. If not, execute the appropriate migration SQL
3. Begin writing data using the Iceberg writer

## Retention

Data retention is set to 1 month (2592000000 milliseconds) via Iceberg's snapshot expiration. This mirrors the ClickHouse TTL policy.