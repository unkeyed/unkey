-- name: ListPrivateNetworkPreviewDeploymentIds :many
-- ListPrivateNetworkPreviewDeploymentIds returns, for each connection from one
-- of the given caller deployments to one of the given apps, the newest
-- deployment of the target app that was ever ready on the caller's git branch
-- and fork in a preview environment. Rows are keyed by target app, branch, and
-- fork; keys without such a deployment have no row.
-- Each key costs one probe of app_branch_fork_ready_created_idx, so the cost
-- does not grow with branch history. The ORDER BY names every index column
-- because MySQL does not treat columns bound by a correlated equality as
-- constant and would otherwise sort the whole branch history per key.
SELECT
    k.workspace_id,
    k.project_id,
    k.app_id,
    k.git_branch,
    k.fork_key,
    (
        SELECT candidate.id
        FROM deployments candidate FORCE INDEX (app_branch_fork_ready_created_idx)
        INNER JOIN environments candidate_env ON candidate_env.id = candidate.environment_id
            AND candidate_env.app_id = candidate.app_id
        WHERE candidate.app_id = k.app_id
            AND candidate.git_branch = k.git_branch
            AND (CASE WHEN candidate.fork_repository_full_name IS NULL THEN '' ELSE candidate.fork_repository_full_name END) = k.fork_key
            AND (candidate.first_ready_at IS NOT NULL) = TRUE
            AND candidate.workspace_id = k.workspace_id
            AND candidate.project_id = k.project_id
            AND candidate.source = 'git'
            AND candidate_env.kind = 'preview'
        ORDER BY
            candidate.app_id DESC,
            candidate.git_branch DESC,
            (CASE WHEN candidate.fork_repository_full_name IS NULL THEN '' ELSE candidate.fork_repository_full_name END) DESC,
            (candidate.first_ready_at IS NOT NULL) DESC,
            candidate.created_at DESC,
            candidate.id DESC
        LIMIT 1
    ) AS id
FROM (
    SELECT DISTINCT
        b.workspace_id,
        b.project_id,
        b.resource_id AS app_id,
        COALESCE(caller.git_branch, '') AS git_branch,
        COALESCE(caller.fork_repository_full_name, '') AS fork_key
    FROM deployment_connections b
    INNER JOIN deployments caller ON caller.id = b.deployment_id
    WHERE b.deployment_id IN (sqlc.slice(caller_deployment_ids))
        AND b.resource_type = 'app'
        AND b.resource_id IN (sqlc.slice(app_ids))
) AS k
HAVING id IS NOT NULL;
