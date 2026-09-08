-- Step 1 of the frontline_latency -> gateway_latency rename. The DEFAULT keeps
-- old writers and old parts correct until the drop; MATERIALIZE writes the
-- bytes into those parts, so the drop must wait for the mutation to finish.

ALTER TABLE `default`.`frontline_requests_raw_v1`
  ADD COLUMN `gateway_latency` Int64 DEFAULT `frontline_latency` AFTER `frontline_latency`;

ALTER TABLE `default`.`frontline_requests_raw_v1`
  MATERIALIZE COLUMN `gateway_latency`;
