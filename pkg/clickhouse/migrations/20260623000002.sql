-- Drop legacy sentinel request storage after services have moved to
-- frontline_requests_* reads and writes and historical raw rows have been
-- backfilled.

DROP VIEW IF EXISTS `default`.`sentinel_requests_per_hour_mv_v1`;
DROP VIEW IF EXISTS `default`.`sentinel_requests_per_15m_mv_v1`;
DROP TABLE IF EXISTS `default`.`sentinel_requests_per_hour_v1`;
DROP TABLE IF EXISTS `default`.`sentinel_requests_per_15m_v1`;
DROP TABLE IF EXISTS `default`.`sentinel_requests_raw_v1`;
