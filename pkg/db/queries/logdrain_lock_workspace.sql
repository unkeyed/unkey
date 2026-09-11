-- name: LockWorkspaceLogdrains :many
-- Current reads prevent stale snapshots after acquiring the workspace limits lock.
-- Create uses this count under the lock shared with dashboard creation.
SELECT id FROM logdrains WHERE workspace_id = sqlc.arg(workspace_id) FOR UPDATE;
