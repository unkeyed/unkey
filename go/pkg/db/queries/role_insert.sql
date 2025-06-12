-- name: InsertRole :exec
INSERT INTO roles (
  id,
  workspace_Id,
  name,
  description,
  created_at_m
)
VALUES (
  sqlc.arg(role_id),
  sqlc.arg(workspace_id),
  sqlc.arg(name),
  sqlc.arg(description),
  sqlc.arg(created_at)
)
;
