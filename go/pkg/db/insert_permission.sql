-- name: InsertPermission :one
-- Inserts a new permission record
-- Returns: The newly created permission record
INSERT INTO "permissions" (
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