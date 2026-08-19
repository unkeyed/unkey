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

-- A durable build ID is written only by the Git build path.
UPDATE `deployments`
SET `source` = 'git'
WHERE `source` = 'unknown'
  AND `build_id` IS NOT NULL
  AND `build_id` <> '';

-- A deployment is safely identifiable as Docker-sourced when it has no Depot
-- build ID and its image is outside Unkey's Depot registry. Both checks are
-- required: failed Git builds can lack a build ID, while old Git redeployments
-- can reuse a Depot image without copying the original build ID.
UPDATE `deployments`
SET `source` = 'docker'
WHERE `source` = 'unknown'
  AND (`build_id` IS NULL OR `build_id` = '')
  AND COALESCE(NULLIF(`requested_image`, ''), NULLIF(`image`, ''), NULLIF(`resolved_image`, '')) IS NOT NULL
  AND COALESCE(NULLIF(`requested_image`, ''), NULLIF(`image`, ''), NULLIF(`resolved_image`, '')) NOT LIKE 'registry.depot.dev/%';

-- Use the newest safely classified Docker deployment as the app's configured
-- default image. Repository-connected apps remain Git-sourced even if they
-- previously received an explicit Docker deployment.
INSERT INTO `app_docker_sources` (
  `workspace_id`,
  `app_id`,
  `image_reference`,
  `created_at`,
  `updated_at`
)
SELECT
  `app`.`workspace_id`,
  `app`.`id`,
  `candidate`.`image_reference`,
  `candidate`.`created_at`,
  `candidate`.`updated_at`
FROM `apps` AS `app`
INNER JOIN (
  SELECT
    `deployment`.`app_id`,
    COALESCE(NULLIF(`deployment`.`requested_image`, ''), NULLIF(`deployment`.`image`, ''), NULLIF(`deployment`.`resolved_image`, '')) AS `image_reference`,
    `deployment`.`created_at`,
    `deployment`.`updated_at`,
    ROW_NUMBER() OVER (
      PARTITION BY `deployment`.`app_id`
      ORDER BY `deployment`.`created_at` DESC, `deployment`.`pk` DESC
    ) AS `row_number`
  FROM `deployments` AS `deployment`
  WHERE `deployment`.`source` = 'docker'
    -- A ready deployment proves the registry accepted and resolved the
    -- reference. Failed or interrupted deployments are not source defaults.
    AND `deployment`.`status` = 'ready'
) AS `candidate` ON `candidate`.`app_id` = `app`.`id` AND `candidate`.`row_number` = 1
LEFT JOIN `github_repo_connections` AS `connection` ON `connection`.`app_id` = `app`.`id`
LEFT JOIN `app_docker_sources` AS `docker_source` ON `docker_source`.`app_id` = `app`.`id`
WHERE `app`.`source_type` = 'unknown'
  AND `connection`.`app_id` IS NULL
  AND `docker_source`.`app_id` IS NULL
  -- New Docker app sources require a complete explicit tag or SHA-256 digest.
  -- Leave implicit or malformed historical references unknown; SQL cannot
  -- safely reproduce full OCI reference normalization.
  AND (
    `candidate`.`image_reference`
      REGEXP '^(([a-z0-9]+([.-][a-z0-9]+)*)(:[0-9]+)?/)?([a-z0-9]+([._-][a-z0-9]+)*/)*[a-z0-9]+([._-][a-z0-9]+)*@sha256:[0-9a-fA-F]{64}$'
    OR `candidate`.`image_reference`
      REGEXP '^(([a-z0-9]+([.-][a-z0-9]+)*)(:[0-9]+)?/)?([a-z0-9]+([._-][a-z0-9]+)*/)*[a-z0-9]+([._-][a-z0-9]+)*:[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$'
  );

UPDATE `apps` AS `app`
INNER JOIN `app_docker_sources` AS `docker_source` ON `docker_source`.`app_id` = `app`.`id`
LEFT JOIN `github_repo_connections` AS `connection` ON `connection`.`app_id` = `app`.`id`
SET `app`.`source_type` = 'docker'
WHERE `app`.`source_type` = 'unknown'
  AND `connection`.`app_id` IS NULL;

-- Backfill the additive resolved-image column. Verify no legacy-only values
-- remain before removing the image column in a later schema change.
UPDATE `deployments`
SET `resolved_image` = `image`
WHERE `image` IS NOT NULL
  AND NOT (`resolved_image` <=> `image`);
