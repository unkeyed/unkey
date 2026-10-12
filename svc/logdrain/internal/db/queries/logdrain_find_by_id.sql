-- FindLogdrainByID returns one drain regardless of status or lease. Only tests
-- call it to observe delivery state.
-- name: FindLogdrainByID :one
SELECT
  id,
  workspace_id,
  name,
  stream,
  config,
  status,
  consecutive_failures,
  committed_offset_inserted_at,
  committed_offset_event_id,
  next_attempt_at,
  lease_id,
  fencing_token,
  lease_expires_at,
  created_at,
  updated_at
FROM logdrains
WHERE id = sqlc.arg(id);
