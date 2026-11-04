-- name: ListPrivateNetworkEnvironmentDeploymentIds :many
-- ListPrivateNetworkEnvironmentDeploymentIds returns, for each app and
-- environment among the given values, the newest deployment that was ever
-- ready in that environment when it is a preview environment of the app. Rows
-- cover every combination of the given values, so callers keep only the keys
-- they asked for.
SELECT ranked.id, ranked.workspace_id, ranked.project_id, ranked.app_id, ranked.environment_id
FROM (
    SELECT
        candidate.id,
        candidate.workspace_id,
        candidate.project_id,
        candidate.app_id,
        candidate.environment_id,
        ROW_NUMBER() OVER (
            PARTITION BY candidate.app_id, candidate.environment_id, candidate.workspace_id, candidate.project_id
            ORDER BY candidate.created_at DESC, candidate.id DESC
        ) AS position
    FROM deployments candidate FORCE INDEX (app_environment_ready_created_idx)
    INNER JOIN environments target_env ON target_env.id = candidate.environment_id
        AND target_env.app_id = candidate.app_id
    WHERE candidate.app_id IN (sqlc.slice(app_ids))
        AND candidate.environment_id IN (sqlc.slice(environment_ids))
        AND (candidate.first_ready_at IS NOT NULL) = TRUE
        AND target_env.kind = 'preview'
) AS ranked
WHERE ranked.position = 1;
