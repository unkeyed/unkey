-- name: ReassignFrontlineRoute :exec
UPDATE frontline_routes
SET
  deployment_id = sqlc.arg(deployment_id),
  updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
