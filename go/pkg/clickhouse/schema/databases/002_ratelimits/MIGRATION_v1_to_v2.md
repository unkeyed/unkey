# Migration Guide: Raw Ratelimits v1 to v2

This guide explains how to migrate from `raw_ratelimits_v1` to `raw_ratelimits_v2` with zero downtime.

## What's Changed in v2

- ✅ **Latency tracking**: New `latency` column for ratelimit check performance
- ✅ **Partitioning**: Monthly partitions for better performance and lifecycle management
- ✅ **TTL**: Automatic data cleanup after 3 months  
- ✅ **Deduplication**: Protection against duplicate inserts during retries
- ✅ **Better ordering**: Optimized ORDER BY for time-series queries
- ✅ **Latency aggregations**: avg, p75, p99 latency metrics in all aggregation tables

## Migration Steps

### Phase 1: Deploy New Schema (Zero Downtime)

1. **Deploy the new schema files** - This creates v2 tables and temporary sync:
   ```bash
   # The auto-migration will create:
   # - raw_ratelimits_v2 (new table)
   # - All new aggregation tables (minute/hour/day/month v2)
   # - temp_sync_v1_to_v2 (materialized view for live sync)
   ```

2. **Verify deployment** - Check that all new tables exist:
   ```sql
   SHOW TABLES FROM ratelimits LIKE '%v2%';
   ```

### Phase 2: Backfill Historical Data

**⚠️ Important**: The temporary materialized view (`temp_sync_v1_to_v2`) is now active and syncing all NEW data from v1 to v2. You can safely backfill historical data while new writes continue.

Run these queries **manually** to backfill historical data in chunks:

#### Step 1: Find your data range
```sql
SELECT 
  toDateTime(min(time)/1000) as earliest_data,
  toDateTime(max(time)/1000) as latest_data,
  count(*) as total_rows
FROM ratelimits.raw_ratelimits_v1;
```

#### Step 2: Backfill month by month
Replace the date ranges based on your actual data from Step 1:

```sql
-- January 2024 (example - adjust dates)
INSERT INTO ratelimits.raw_ratelimits_v2 
SELECT 
    request_id,
    time,
    workspace_id,
    namespace_id,
    identifier,
    passed,
    0.0 as latency         -- v1 doesn't have this column, default to 0.0
FROM ratelimits.raw_ratelimits_v1 
WHERE time >= toUnixTimestamp64Milli(toDateTime('2024-01-01 00:00:00'))
  AND time < toUnixTimestamp64Milli(toDateTime('2024-02-01 00:00:00'));

-- February 2024
INSERT INTO ratelimits.raw_ratelimits_v2 
SELECT 
    request_id,
    time,
    workspace_id,
    namespace_id,
    identifier,
    passed,
    0.0 as latency
FROM ratelimits.raw_ratelimits_v1 
WHERE time >= toUnixTimestamp64Milli(toDateTime('2024-02-01 00:00:00'))
  AND time < toUnixTimestamp64Milli(toDateTime('2024-03-01 00:00:00'));

-- Continue for each month until you reach current data...
```

#### Step 3: Monitor progress
```sql
-- Check backfill progress
SELECT 
  'v1' as version,
  toDateTime(min(time)/1000) as earliest,
  toDateTime(max(time)/1000) as latest,
  count(*) as rows
FROM ratelimits.raw_ratelimits_v1
UNION ALL
SELECT 
  'v2' as version,
  toDateTime(min(time)/1000) as earliest,
  toDateTime(max(time)/1000) as latest,
  count(*) as rows
FROM ratelimits.raw_ratelimits_v2
ORDER BY version;
```

### Phase 3: Switch Application to v2

1. **Update your application** to write to `raw_ratelimits_v2` instead of `v1`
2. **Include the new field**:
   - `latency` (Float64) - ratelimit check latency in milliseconds with sub-ms precision

### Phase 4: Cleanup (After Confirming v2 Works)

1. **Drop the temporary sync view**:
   ```sql
   DROP VIEW ratelimits.temp_sync_v1_to_v2;
   ```

2. **Optional: Keep v1 as backup** or drop it:
   ```sql
   -- To drop v1 (only after confirming v2 works perfectly)
   DROP TABLE ratelimits.raw_ratelimits_v1;
   ```

## Rollback Plan

If you need to rollback:

1. **Switch application** back to writing to `v1`
2. **Keep the temp sync view** active
3. **Investigate issues** with v2
4. **Re-run migration** when ready

## Important Notes

- 🔄 **Zero downtime**: New data automatically flows to v2 via the temp sync view
- 📊 **Historical data**: Will have `latency=0.0` (can be updated later when you add latency tracking)
- 🗓️ **TTL**: v2 data will auto-delete after 3 months
- 🔍 **Deduplication**: Protects against duplicate inserts during retries
- 📈 **Performance**: Monthly partitions improve query performance on time ranges
- ⚡ **TDigest quantiles**: Efficient approximate percentile calculations for latency

## New Aggregation Metrics Available

With v2, all aggregation tables now include:
- `latency_avg`: Average ratelimit check latency
- `latency_p75`: 75th percentile latency
- `latency_p99`: 99th percentile latency

Perfect for monitoring ratelimit performance and identifying bottlenecks!

## Questions?

- Check the temp sync view is active: `SELECT * FROM system.tables WHERE name = 'temp_sync_v1_to_v2'`
- Monitor both tables during migration to ensure data consistency
- The aggregation tables (minute/hour/day/month) will automatically populate from v2 data