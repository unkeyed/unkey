-- name: ListLiveEnvironmentIdsByApp :many
-- Returns the IDs of environments under an app that are not already
-- scheduled for permanent deletion. Same rationale as the live-apps
-- variant: cascaded soft-delete skips independently-deleted children.
SELECT id
FROM environments
WHERE app_id = sqlc.arg(app_id)
  AND deletion_id IS NULL;
