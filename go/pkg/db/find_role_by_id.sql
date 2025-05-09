-- name: FindRoleById :one
-- Finds a role record by its ID
-- Returns: The role record if found
SELECT *
FROM "roles"
WHERE "id" = $1
LIMIT 1;