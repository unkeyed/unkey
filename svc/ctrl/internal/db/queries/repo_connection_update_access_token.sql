-- name: UpdateRepoConnectionAccessToken :exec
-- POC: persists a rotated clone credential (Bitbucket refresh tokens rotate
-- on every use; storing the stale one trips reuse detection, which revokes
-- the whole token family).
UPDATE github_repo_connections
SET access_token = sqlc.arg(access_token),
    updated_at = sqlc.arg(updated_at)
WHERE app_id = sqlc.arg(app_id);
