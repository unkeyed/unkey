-- name: FindAppByIdAndWorkspace :one
-- Resolves an app by id within a workspace.
--
-- app_find_by_id.sql has no workspace predicate, and the project-scoped finders
-- need a project identifier that callers holding only an app id do not have.
-- Anything validating that a caller owns the app it named must scope the lookup,
-- so this exists as the scoped single-app read.
--
-- The project id comes back with it because an app's resource permissions are
-- addressed as projects/{project_id}/apps/{app_id}: a caller that arrived with
-- an app id alone would otherwise need a second read to say anything about the
-- app it just proved it owns.
SELECT apps.id, apps.project_id FROM apps
WHERE apps.id = sqlc.arg('id')
  AND apps.deleted_at IS NULL
  AND EXISTS (SELECT 1 FROM projects p WHERE p.id = apps.project_id AND p.deleted_at IS NULL)
  AND apps.workspace_id = sqlc.arg('workspace_id');
