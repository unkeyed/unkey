-- name: InsertDeletion :exec
-- Records a resource that has been marked for permanent deletion at
-- delete_permanently_at. The caller mints the id and writes it into
-- the resource's deletion_id column in the same transaction.
--
-- Idempotent on (resource_type, resource_id): if a row already exists
-- (because the resource was independently deleted before this cascade
-- reached it), keep the original row — preserving its id and timestamp
-- so the independently-initiated restore tree stays intact. The no-op
-- ON DUPLICATE form avoids INSERT IGNORE which would also swallow
-- other errors.
INSERT INTO `deletions` (id, workspace_id, resource_type, resource_id, delete_permanently_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(resource_type), sqlc.arg(resource_id), sqlc.arg(delete_permanently_at))
ON DUPLICATE KEY UPDATE resource_id = resource_id;
