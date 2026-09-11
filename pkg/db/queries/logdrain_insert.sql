-- name: InsertLogdrain :exec
-- The caller holds the workspace limits lock and has checked current capacity.
-- Initialize the cursor at creation time so historical records are not exported.
INSERT INTO logdrains (id, workspace_id, name, stream, config, committed_offset_inserted_at, lease_id, fencing_token, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(name), sqlc.arg(stream), sqlc.arg(config), sqlc.arg(created_at), '', '', sqlc.arg(created_at), sqlc.arg(updated_at));
