-- name: InsertRole :one
-- Inserts a new role record
-- Returns: The newly created role record
INSERT INTO "roles" (
  "id",
  "workspaceId",
  "name",
  "description"
)
VALUES (
  $1, -- id
  $2, -- workspaceId
  $3, -- name
  $4  -- description
)
RETURNING *;