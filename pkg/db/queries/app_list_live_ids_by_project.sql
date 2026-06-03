-- name: ListLiveAppIdsByProject :many
-- Returns the IDs of apps under a project that are not already
-- scheduled for permanent deletion. Used by the soft-delete cascade
-- so a parent's SoftDelete does not disturb apps that were deleted
-- independently (those already have their own deletions row with a
-- different T).
SELECT id
FROM apps
WHERE project_id = sqlc.arg(project_id)
  AND deletion_id IS NULL;
