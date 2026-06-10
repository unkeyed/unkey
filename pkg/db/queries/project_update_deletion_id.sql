-- name: UpdateProjectDeletionId :execresult
-- Compare-and-swap on deletion_id. The write only applies when the
-- row's current deletion_id matches expected_deletion_id (NULL-safe
-- via the <=> operator). Same query handles both directions:
--   - Set:   expected=NULL,  new=<id>
--   - Clear: expected=<id>,  new=NULL
-- Callers should check the result's RowsAffected to know whether the
-- swap took effect; a 0 means the row was concurrently mutated by
-- someone else and the caller's view is stale.
UPDATE projects
SET
    deletion_id = sqlc.arg(deletion_id),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND deletion_id <=> sqlc.arg(expected_deletion_id);
