-- name: FindKeysByKeyAuthId :many
-- Finds all active keys (not deleted) for a specific keyAuth
-- Returns: All keys associated with the given keyAuthId
SELECT *
FROM "keys"
WHERE "keyAuthId" = $1
AND "deletedAtM" IS NULL;