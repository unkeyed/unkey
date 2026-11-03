-- Add skip indexes for the path predicate of the API-request log listing.
--
-- default.api_requests_raw_v2 sorts by (workspace_id, time, request_id) and
-- indexed only request_id and inserted_at, so a path predicate could not prune
-- granules: one dashboard page filtered by path scanned the workspace's whole
-- retention slice and timed out. frontline_requests_raw_v1 got these same
-- indexes in 20260814000000.sql, api_requests_raw_v2 never did.
--
-- Exact path predicates use a bloom filter. Prefix and suffix predicates use a
-- trigram index. MATERIALIZE INDEX builds both for existing parts, which
-- otherwise only cover new inserts.

ALTER TABLE `default`.`api_requests_raw_v2`
    ADD INDEX IF NOT EXISTS idx_path path TYPE bloom_filter(0.01) GRANULARITY 1;

ALTER TABLE `default`.`api_requests_raw_v2`
    ADD INDEX IF NOT EXISTS idx_path_text_search path
    TYPE ngrambf_v1(3, 32768, 2, 0) GRANULARITY 1;

ALTER TABLE `default`.`api_requests_raw_v2`
    MATERIALIZE INDEX idx_path;

ALTER TABLE `default`.`api_requests_raw_v2`
    MATERIALIZE INDEX idx_path_text_search;
