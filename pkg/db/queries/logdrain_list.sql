-- name: ListLogdrains :many
-- Stable ID ordering supports pagination without crossing the workspace boundary.
-- The inclusive cursor is the first candidate of the next page.
SELECT pk, id, workspace_id, name, stream, config, status, consecutive_failures,
  committed_offset_inserted_at, committed_offset_event_id, next_attempt_at,
  lease_id, fencing_token, lease_expires_at, created_at, updated_at
FROM logdrains WHERE workspace_id = sqlc.arg(workspace_id) AND id >= sqlc.arg(id_cursor)
ORDER BY id ASC LIMIT ?;
