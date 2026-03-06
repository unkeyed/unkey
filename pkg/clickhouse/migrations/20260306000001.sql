-- Drop the v2 key verification aggregation chain from default database.
-- The v3 chain fully replaces it with better partitioning and longer TTL.
-- Order: leaf MVs → upstream MVs → tables

-- 1. Drop v2 aggregation MVs (leaf → root)
DROP VIEW IF EXISTS `default`.`key_verifications_per_month_mv_v2`;
DROP VIEW IF EXISTS `default`.`key_verifications_per_day_mv_v2`;
DROP VIEW IF EXISTS `default`.`key_verifications_per_hour_mv_v2`;
DROP VIEW IF EXISTS `default`.`key_verifications_per_minute_mv_v2`;

-- 2. Drop v2 aggregation tables
DROP TABLE IF EXISTS `default`.`key_verifications_per_month_v2`;
DROP TABLE IF EXISTS `default`.`key_verifications_per_day_v2`;
DROP TABLE IF EXISTS `default`.`key_verifications_per_hour_v2`;
DROP TABLE IF EXISTS `default`.`key_verifications_per_minute_v2`;
