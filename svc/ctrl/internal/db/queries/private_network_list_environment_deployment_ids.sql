-- name: ListPrivateNetworkEnvironmentDeploymentIds :many
-- ListPrivateNetworkEnvironmentDeploymentIds returns, for each given preview
-- environment, the newest deployment that was ever ready in it. Environments
-- without such a deployment have no row. Each environment costs one probe of
-- app_environment_ready_created_idx, so the cost does not grow with history.
SELECT
    e.workspace_id,
    e.project_id,
    e.app_id,
    e.id AS environment_id,
    newest.id
FROM environments e
INNER JOIN deployments newest ON newest.id = (
    SELECT candidate.id
    FROM deployments candidate FORCE INDEX (app_environment_ready_created_idx)
    WHERE candidate.app_id = e.app_id
        AND candidate.environment_id = e.id
        AND (candidate.first_ready_at IS NOT NULL) = TRUE
        AND candidate.workspace_id = e.workspace_id
        AND candidate.project_id = e.project_id
    ORDER BY candidate.created_at DESC, candidate.id DESC
    LIMIT 1
)
WHERE e.id IN (sqlc.slice(environment_ids))
    AND e.kind = 'preview';
