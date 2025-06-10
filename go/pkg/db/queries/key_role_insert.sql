-- name: InsertKeyRole :exec
INSERT INTO keys_roles (
  key_id,
  role_id,
  workspace_id,
  created_at_m
)
VALUES (
  sqlc.arg(key_id),
  sqlc.arg(role_id),
  sqlc.arg(workspace_id),
  sqlc.arg(created_at_m)
);