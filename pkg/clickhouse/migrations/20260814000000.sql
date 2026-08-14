-- Add skip indexes for the request-log filters. Exact host and path predicates
-- use bloom filters. Region uses a bounded set index. Path prefix
-- and substring predicates and case-insensitive user-agent substring predicates
-- use trigram indexes that match the dashboard query expressions.

ALTER TABLE `default`.`frontline_requests_raw_v1`
    ADD INDEX IF NOT EXISTS idx_host host TYPE bloom_filter(0.01) GRANULARITY 1;

ALTER TABLE `default`.`frontline_requests_raw_v1`
    ADD INDEX IF NOT EXISTS idx_path path TYPE bloom_filter(0.01) GRANULARITY 1;

ALTER TABLE `default`.`frontline_requests_raw_v1`
    ADD INDEX IF NOT EXISTS idx_path_text_search path
    TYPE ngrambf_v1(3, 32768, 2, 0) GRANULARITY 1;

ALTER TABLE `default`.`frontline_requests_raw_v1`
    ADD INDEX IF NOT EXISTS idx_region region TYPE set(64) GRANULARITY 1;

ALTER TABLE `default`.`frontline_requests_raw_v1`
    ADD INDEX IF NOT EXISTS idx_user_agent_text_search lower(user_agent)
    TYPE ngrambf_v1(3, 32768, 2, 0) GRANULARITY 1;
