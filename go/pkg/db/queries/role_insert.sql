-- name: InsertRole :exec
INSERT INTO roles (
  id,
  workspace_Id,
  name,
  description
)
VALUES (
  sqlc.arg(role_id),
  sqlc.arg(workspace_id),
  sqlc.arg(name),
  sqlc.arg(description)
)
;
