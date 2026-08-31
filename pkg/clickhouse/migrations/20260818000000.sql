-- Step 2 of the frontline_latency -> gateway_latency rename.
ALTER TABLE `default`.`frontline_requests_raw_v1`
  MODIFY COLUMN `gateway_latency` REMOVE DEFAULT;

ALTER TABLE `default`.`frontline_requests_raw_v1`
  DROP COLUMN `frontline_latency`;
