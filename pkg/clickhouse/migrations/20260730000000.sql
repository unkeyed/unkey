-- Add a narrow projection for project-wide sentinel log pagination. The base
-- table is ordered by app and environment before time, so a request for the
-- newest logs across an entire project must scan every app/environment lane.
-- This projection covers every column used by the page-selection filters and
-- orders rows by project time without duplicating request/response payloads.
--
-- ADD PROJECTION only affects parts written after this migration. Do not run
-- MATERIALIZE PROJECTION: existing parts live on S3-backed storage and expire
-- within the table's 7-day TTL, so rewriting them is costly and unnecessary.

ALTER TABLE `default`.`frontline_requests_raw_v1`
ADD PROJECTION IF NOT EXISTS `p_logs_by_project_time` (
  SELECT
    `workspace_id`,
    `project_id`,
    `time`,
    `request_id`,
    `deployment_id`,
    `environment_id`,
    `response_status`,
    `method`,
    `path`
  ORDER BY (`workspace_id`, `project_id`, `time`, `request_id`)
);
