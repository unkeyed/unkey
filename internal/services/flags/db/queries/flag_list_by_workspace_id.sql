-- name: ListFlagsByWorkspaceID :many
-- ListFlagsByWorkspaceID returns every definition with the workspace's
-- override. The LEFT JOIN keeps definitions without an override as a NULL
-- override_value, so callers can tell inheritance apart from an explicit false.
-- Overrides whose definition was deleted are not returned.
SELECT f.slug, f.default_value, o.value AS override_value
FROM flags f
LEFT JOIN flag_overrides o
    ON o.flag_slug = f.slug AND o.workspace_id = sqlc.arg(workspace_id);
