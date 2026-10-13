-- name: ListFlags :many
-- ListFlags includes definitions without overrides. Callers resolve NULL override
-- values to the default, preserving explicit false overrides.
SELECT f.pk, f.slug, f.description, f.default_value,
    f.allow_enable, f.allow_disable, o.value AS override_value
FROM flags f
LEFT JOIN flag_overrides o
    ON o.flag_slug = f.slug AND o.workspace_id = sqlc.arg(workspace_id)
ORDER BY f.slug;
