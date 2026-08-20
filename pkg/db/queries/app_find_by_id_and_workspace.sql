-- name: FindAppByIdAndWorkspace :one
-- Resolves an app by id within a workspace.
--
-- app_find_by_id.sql has no workspace predicate, and the project-scoped finders
-- need a project identifier that callers holding only an app id do not have.
-- Anything validating that a caller owns the app it named must scope the lookup,
-- so this exists as the scoped single-app read.
SELECT * FROM apps
WHERE id = sqlc.arg('id')
  AND workspace_id = sqlc.arg('workspace_id');
