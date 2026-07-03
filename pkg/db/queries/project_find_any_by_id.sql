-- name: FindProjectAnyById :one
-- Returns a project row by id without filtering on delete_permanently_at.
-- Used by DeleteProject/RestoreProject to load the row for response
-- composition; the normal FindProjectById hides scheduled-for-deletion
-- rows from the API path.
SELECT *
FROM projects
WHERE id = ?;
