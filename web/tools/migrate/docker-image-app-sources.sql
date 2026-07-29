-- Run after the additive app/deployment source schema is deployed.
-- Both updates are idempotent and intentionally leave ambiguous rows alone.

UPDATE `github_repo_connections` AS `connection`
INNER JOIN `apps` AS `app` ON `app`.`id` = `connection`.`app_id`
SET `connection`.`default_branch` = NULLIF(`app`.`default_branch`, '')
WHERE `connection`.`default_branch` IS NULL;

-- A durable build ID is written only by the Git build path. Do not infer
-- Docker provenance from its absence: failed Git builds can lack one, and old
-- image redeployments may carry copied Git metadata.
UPDATE `deployments`
SET `source` = 'git_build'
WHERE `source` = 'unknown'
  AND `build_id` IS NOT NULL
  AND `build_id` <> '';
