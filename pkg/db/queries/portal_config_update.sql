-- name: UpdatePortalConfig :exec
-- Updates a portal configuration's mutable fields, scoped to the workspace so
-- one workspace can never mutate another's config. A dedicated scoped UPDATE
-- (rather than an upsert) is used because portal_configurations has four unique
-- keys (id, workspace_id+slug, app_id, key_auth_id): ON DUPLICATE KEY UPDATE
-- across multiple unique indexes has undefined row selection and could silently
-- mutate the wrong row on a collision. Here a slug/app/keyspace collision with a
-- different row surfaces the unique-constraint error, which the handler maps to
-- a 409.
UPDATE portal_configurations
SET
    slug = sqlc.arg(slug),
    app_id = sqlc.narg(app_id),
    key_auth_id = sqlc.narg(key_auth_id),
    enabled = sqlc.arg(enabled),
    return_url = sqlc.narg(return_url),
    updated_at = sqlc.narg(updated_at)
WHERE id = sqlc.arg(id) AND workspace_id = sqlc.arg(workspace_id);
