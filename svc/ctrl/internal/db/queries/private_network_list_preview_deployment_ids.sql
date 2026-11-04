-- name: ListPrivateNetworkPreviewDeploymentIds :many
-- ListPrivateNetworkPreviewDeploymentIds returns, for each app, git branch,
-- and fork among the given values, the newest deployment that was ever ready
-- on that branch and fork in a preview environment of the app. Rows cover
-- every combination of the given values, so callers keep only the keys they
-- asked for.
-- Without the hint MySQL may drive from the app's preview environments
-- through app_environment_ready_created_idx and sort the whole branch.
SELECT ranked.id, ranked.workspace_id, ranked.project_id, ranked.app_id, ranked.git_branch, ranked.fork_key
FROM (
    SELECT
        candidate.id,
        candidate.workspace_id,
        candidate.project_id,
        candidate.app_id,
        COALESCE(candidate.git_branch, '') AS git_branch,
        COALESCE(candidate.fork_repository_full_name, '') AS fork_key,
        ROW_NUMBER() OVER (
            PARTITION BY candidate.app_id, candidate.git_branch,
                (CASE WHEN candidate.fork_repository_full_name IS NULL THEN '' ELSE candidate.fork_repository_full_name END),
                candidate.workspace_id, candidate.project_id
            ORDER BY candidate.created_at DESC, candidate.id DESC
        ) AS position
    FROM deployments candidate FORCE INDEX (app_branch_fork_ready_created_idx)
    INNER JOIN environments candidate_env ON candidate_env.id = candidate.environment_id
        AND candidate_env.app_id = candidate.app_id
    WHERE candidate.app_id IN (sqlc.slice(app_ids))
        AND candidate.git_branch IN (sqlc.slice(git_branches))
        AND (candidate.first_ready_at IS NOT NULL) = TRUE
        AND candidate.source = 'git'
        AND candidate_env.kind = 'preview'
) AS ranked
WHERE ranked.position = 1;
