-- name: ListFlagsByWorkspaceID :many
-- ListFlagsByWorkspaceID returns every definition with the workspace's
-- override. The LEFT JOIN keeps definitions without an override as a NULL
-- override_enabled, so callers can tell inheritance apart from an explicit false.
-- Overrides whose definition was deleted are not returned.
SELECT f.slug, f.default_enabled, f.description, f.allow_opt_in, f.allow_opt_out, o.enabled AS override_enabled
FROM flags f
LEFT JOIN flag_overrides o
    ON o.flag_slug = f.slug AND o.workspace_id = sqlc.arg(workspace_id);
