-- name: FindAppByIdAndWorkspace :one
-- FindAppByIdAndWorkspace resolves an app by ID within a workspace.
--
-- app_find_by_id.sql has no workspace predicate, and the project-scoped finders
-- need a project identifier that callers holding only an app ID do not have.
-- Anything validating that a caller owns the app it named must scope the lookup,
-- so this exists as the scoped single-app read.
--
-- The project ID lets callers build project-scoped resource permissions after
-- they verify workspace ownership.
SELECT id, project_id FROM apps
WHERE id = sqlc.arg('id')
  AND workspace_id = sqlc.arg('workspace_id');
