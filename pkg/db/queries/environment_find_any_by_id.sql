-- name: FindEnvironmentAnyById :one
-- Returns an environment row by id without filtering on
-- delete_permanently_at. Used by SoftDelete/Restore VOs.
SELECT *
FROM environments
WHERE id = sqlc.arg(id);
