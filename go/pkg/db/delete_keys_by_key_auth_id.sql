-- name: DeleteKeysByKeyAuthId :execresult
-- Soft deletes all keys for a specific keyAuth by setting their deletedAtM field
-- Returns: Result of the update operation
UPDATE "keys"
SET "deletedAtM" = $2
WHERE "keyAuthId" = $1
AND "deletedAtM" IS NULL;