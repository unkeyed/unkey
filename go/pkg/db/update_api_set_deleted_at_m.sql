-- name: UpdateApiSetDeletedAtM :one
-- Updates the deletedAtM field of an API to mark it as deleted
-- Returns: The updated API record
UPDATE "apis"
SET "deletedAtM" = $2
WHERE "id" = $1
RETURNING *;