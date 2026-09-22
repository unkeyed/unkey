-- Requests the gateway rejects before reaching an instance (key auth, rate
-- limit, firewall, OpenAPI validation) are now written to the request log.
-- error_code carries the Unkey error URN so a rejection can be told apart
-- from an upstream 4xx and attributed to the policy that produced it.
-- Existing rows read as empty, which matches a request an instance served.
ALTER TABLE `default`.`frontline_requests_raw_v1`
    ADD COLUMN `error_code` LowCardinality(String) AFTER `gateway_latency`;
