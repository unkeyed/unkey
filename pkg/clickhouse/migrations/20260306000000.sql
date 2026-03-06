-- Drop all legacy ClickHouse tables, materialized views, and databases.
-- The v2/v3 schema in the `default` database fully replaces these.
--
-- Order: sync MVs → leaf MVs → upstream MVs → tables → databases
-- Each section drops MVs before their source/target tables.

-- ============================================================
-- 1. Drop temp sync MVs (bridge v1 raw → v2 raw in default db)
-- ============================================================
DROP VIEW IF EXISTS `default`.`temp_sync_key_verifications_v1_to_v2`;
DROP VIEW IF EXISTS `default`.`temp_sync_ratelimits_raw_v1_to_v2`;
DROP VIEW IF EXISTS `default`.`temp_sync_metrics_v1_to_v2`;

-- ============================================================
-- 2. Drop legacy billing MVs (read from legacy aggregation tables)
-- ============================================================
DROP VIEW IF EXISTS `billing`.`billable_verifications_per_month_mv_v1`;
DROP VIEW IF EXISTS `billing`.`billable_verifications_per_month_mv_v2`;
DROP VIEW IF EXISTS `billing`.`billable_ratelimits_per_month_mv_v1`;

-- ============================================================
-- 3. Drop legacy business MVs
-- ============================================================
DROP VIEW IF EXISTS `business`.`active_workspaces_keys_per_month_mv_v1`;
DROP VIEW IF EXISTS `business`.`active_workspaces_ratelimits_per_month_mv_v1`;

-- ============================================================
-- 4. Drop legacy verifications MVs (all versions in legacy db)
-- ============================================================
DROP VIEW IF EXISTS `verifications`.`key_verifications_per_month_mv_v3`;
DROP VIEW IF EXISTS `verifications`.`key_verifications_per_month_mv_v2`;
DROP VIEW IF EXISTS `verifications`.`key_verifications_per_month_mv_v1`;
DROP VIEW IF EXISTS `verifications`.`key_verifications_per_day_mv_v3`;
DROP VIEW IF EXISTS `verifications`.`key_verifications_per_day_mv_v2`;
DROP VIEW IF EXISTS `verifications`.`key_verifications_per_day_mv_v1`;
DROP VIEW IF EXISTS `verifications`.`key_verifications_per_hour_mv_v3`;
DROP VIEW IF EXISTS `verifications`.`key_verifications_per_hour_mv_v2`;
DROP VIEW IF EXISTS `verifications`.`key_verifications_per_hour_mv_v1`;
DROP VIEW IF EXISTS `verifications`.`key_verifications_per_minute_mv_v1`;

-- ============================================================
-- 5. Drop legacy ratelimits MVs
-- ============================================================
DROP VIEW IF EXISTS `ratelimits`.`ratelimits_per_month_mv_v1`;
DROP VIEW IF EXISTS `ratelimits`.`ratelimits_per_day_mv_v1`;
DROP VIEW IF EXISTS `ratelimits`.`ratelimits_per_hour_mv_v1`;
DROP VIEW IF EXISTS `ratelimits`.`ratelimits_per_minute_mv_v1`;
DROP VIEW IF EXISTS `ratelimits`.`ratelimits_last_used_mv_v1`;

-- ============================================================
-- 6. Drop legacy metrics MVs
-- ============================================================
DROP VIEW IF EXISTS `metrics`.`api_requests_per_day_mv_v1`;
DROP VIEW IF EXISTS `metrics`.`api_requests_per_hour_mv_v1`;
DROP VIEW IF EXISTS `metrics`.`api_requests_per_minute_mv_v1`;

-- ============================================================
-- 7. Drop legacy verifications tables
-- ============================================================
DROP TABLE IF EXISTS `verifications`.`key_verifications_per_month_v3`;
DROP TABLE IF EXISTS `verifications`.`key_verifications_per_month_v2`;
DROP TABLE IF EXISTS `verifications`.`key_verifications_per_month_v1`;
DROP TABLE IF EXISTS `verifications`.`key_verifications_per_day_v3`;
DROP TABLE IF EXISTS `verifications`.`key_verifications_per_day_v2`;
DROP TABLE IF EXISTS `verifications`.`key_verifications_per_day_v1`;
DROP TABLE IF EXISTS `verifications`.`key_verifications_per_hour_v3`;
DROP TABLE IF EXISTS `verifications`.`key_verifications_per_hour_v2`;
DROP TABLE IF EXISTS `verifications`.`key_verifications_per_hour_v1`;
DROP TABLE IF EXISTS `verifications`.`key_verifications_per_minute_v1`;
DROP TABLE IF EXISTS `verifications`.`raw_key_verifications_v1`;

-- ============================================================
-- 8. Drop legacy ratelimits tables
-- ============================================================
DROP TABLE IF EXISTS `ratelimits`.`ratelimits_per_month_v1`;
DROP TABLE IF EXISTS `ratelimits`.`ratelimits_per_day_v1`;
DROP TABLE IF EXISTS `ratelimits`.`ratelimits_per_hour_v1`;
DROP TABLE IF EXISTS `ratelimits`.`ratelimits_per_minute_v1`;
DROP TABLE IF EXISTS `ratelimits`.`ratelimits_last_used_v1`;
DROP TABLE IF EXISTS `ratelimits`.`raw_ratelimits_v1`;

-- ============================================================
-- 9. Drop legacy metrics tables
-- ============================================================
DROP TABLE IF EXISTS `metrics`.`api_requests_per_day_v1`;
DROP TABLE IF EXISTS `metrics`.`api_requests_per_hour_v1`;
DROP TABLE IF EXISTS `metrics`.`api_requests_per_minute_v1`;
DROP TABLE IF EXISTS `metrics`.`raw_api_requests_v1`;

-- ============================================================
-- 10. Drop legacy billing tables
-- ============================================================
DROP TABLE IF EXISTS `billing`.`billable_verifications_per_month_v1`;
DROP TABLE IF EXISTS `billing`.`billable_verifications_per_month_v2`;
DROP TABLE IF EXISTS `billing`.`billable_ratelimits_per_month_v1`;

-- ============================================================
-- 11. Drop legacy business tables
-- ============================================================
DROP TABLE IF EXISTS `business`.`active_workspaces_per_month_v1`;

-- ============================================================
-- 12. Drop legacy databases
-- ============================================================
DROP DATABASE IF EXISTS `verifications`;
DROP DATABASE IF EXISTS `ratelimits`;
DROP DATABASE IF EXISTS `metrics`;
DROP DATABASE IF EXISTS `billing`;
DROP DATABASE IF EXISTS `business`;
