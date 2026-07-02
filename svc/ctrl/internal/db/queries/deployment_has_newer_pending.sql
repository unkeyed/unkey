-- name: HasNewerActiveDeployment :one
-- Check whether a newer deployment exists for the same (app, env, branch) that
-- makes building this one pointless. Matches any non-terminal status including
-- 'ready' — if a newer commit is already deployed there is no reason to build
-- an older one.
--
-- Uses MySQL's NULL-safe equal (<=>) on git_branch so docker-image-only
-- deployments (where git_branch IS NULL on both rows) still detect each other
-- as siblings. Standard `=` returns UNKNOWN for NULL=NULL, which would silently
-- bypass the guardrail for non-git apps.
SELECT EXISTS (
    SELECT 1 FROM deployments
    WHERE app_id = sqlc.arg('app_id')
      AND environment_id = sqlc.arg('environment_id')
      AND git_branch <=> sqlc.arg('git_branch')
      AND status NOT IN ('failed', 'skipped', 'stopped', 'superseded', 'cancelled')
      AND created_at > sqlc.arg('created_at')
      AND id != sqlc.arg('deployment_id')
) AS has_newer;
