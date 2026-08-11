-- Run after the additive schema and dual-writing binaries are deployed, and
-- after older binaries that only write deployments.image have drained.
-- These updates are idempotent and intentionally leave ambiguous rows alone.

UPDATE `github_repo_connections` AS `connection`
INNER JOIN `apps` AS `app` ON `app`.`id` = `connection`.`app_id`
SET `connection`.`default_branch` = COALESCE(NULLIF(`app`.`default_branch`, ''), 'main')
WHERE `connection`.`default_branch` IS NULL;

-- A repository connection unambiguously identifies an existing app as Git-sourced.
UPDATE `apps` AS `app`
INNER JOIN `github_repo_connections` AS `connection` ON `connection`.`app_id` = `app`.`id`
SET `app`.`source_type` = 'git'
WHERE `app`.`source_type` = 'unknown';

-- A durable build ID is written only by the Git build path. Do not infer
-- Docker provenance from its absence: failed Git builds can lack one, and old
-- image redeployments may carry copied Git metadata.
UPDATE `deployments`
SET `source` = 'git'
WHERE `source` = 'unknown'
  AND `build_id` IS NOT NULL
  AND `build_id` <> '';

-- Backfill the additive resolved-image column. Verify no legacy-only values
-- remain before removing the image column in a later schema change.
UPDATE `deployments`
SET `resolved_image` = `image`
WHERE `image` IS NOT NULL
  AND NOT (`resolved_image` <=> `image`);
