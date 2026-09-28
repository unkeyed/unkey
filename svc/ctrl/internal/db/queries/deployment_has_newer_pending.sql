-- name: HasNewerActiveDeployment :one
-- Check whether a newer deployment exists for the same (app, env, branch) that
-- makes building this one pointless. Matches any non-terminal status including
-- 'ready' — if a newer commit is already deployed there is no reason to build
-- an older one.
--
-- Uses MySQL's NULL-safe equal (<=>) on git_branch so OCI-only
-- deployments (where git_branch IS NULL on both rows) still detect each other
-- as siblings. Standard `=` returns UNKNOWN for NULL=NULL, which would silently
-- bypass the guardrail for non-git apps.
SELECT EXISTS (
    SELECT 1 FROM deployments d
    WHERE d.app_id = sqlc.arg('app_id')
      AND d.environment_id = sqlc.arg('environment_id')
      AND d.git_branch <=> sqlc.arg('git_branch')
      AND d.status NOT IN ('failed', 'skipped', 'stopped', 'superseded', 'cancelled')
      AND d.created_at > sqlc.arg('created_at')
      AND d.id != sqlc.arg('deployment_id')
) AS has_newer;
