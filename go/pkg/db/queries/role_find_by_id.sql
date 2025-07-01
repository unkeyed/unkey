-- name: FindRoleByID :one
-- Finds a role record by its ID
-- Returns: The role record if found
SELECT *
FROM roles
WHERE id = sqlc.arg(role_id)
LIMIT 1;
