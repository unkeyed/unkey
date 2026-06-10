-- name: UpdateAppDeletionId :execresult
-- CAS on deletion_id. See project_update_deletion_id.sql.
UPDATE apps
SET
    deletion_id = sqlc.arg(deletion_id),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND deletion_id <=> sqlc.arg(expected_deletion_id);
