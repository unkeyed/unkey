-- name: ListFlags :many
-- ListFlags includes definitions without overrides. Callers resolve NULL override
-- values to the default, preserving explicit false overrides.
SELECT f.pk, f.slug, f.description, f.default_enabled,
    f.allow_opt_in, f.allow_opt_out, o.enabled AS override_enabled
FROM flags f
LEFT JOIN flag_overrides o
    ON o.flag_slug = f.slug AND o.workspace_id = sqlc.arg(workspace_id)
ORDER BY f.slug;
