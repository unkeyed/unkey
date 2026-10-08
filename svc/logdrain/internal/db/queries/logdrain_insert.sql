-- InsertLogdrain creates a drain with every delivery-state column set
-- explicitly. Only tests call it: the dashboard creates drains in production.
-- name: InsertLogdrain :exec
INSERT INTO logdrains (
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
  created_at
)
VALUES (
  sqlc.arg(id),
  sqlc.arg(workspace_id),
  sqlc.arg(name),
  sqlc.arg(stream),
  sqlc.arg(config),
  sqlc.arg(status),
  sqlc.arg(consecutive_failures),
  sqlc.arg(committed_offset_inserted_at),
  sqlc.arg(committed_offset_event_id),
  sqlc.arg(next_attempt_at),
  sqlc.arg(lease_id),
  sqlc.arg(fencing_token),
  sqlc.arg(lease_expires_at),
  sqlc.arg(created_at)
);
