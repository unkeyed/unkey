-- name: DeletePermission :execresult
-- Deletes a permission by its ID
-- Returns: Result of the delete operation
DELETE FROM "permissions"
WHERE "id" = $1;