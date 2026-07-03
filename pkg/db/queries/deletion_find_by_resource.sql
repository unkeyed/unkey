-- name: FindDeletionByResource :one
-- Returns the deletion row for a given resource. Used by Restore to
-- read the cascade-correlation timestamp T before walking children.
SELECT *
FROM `deletions`
WHERE resource_type = sqlc.arg(resource_type)
  AND resource_id = sqlc.arg(resource_id);
