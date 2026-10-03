-- name: ListFlags :many
-- ListFlags includes definitions without overrides. Callers resolve NULL override
-- values to the default, preserving explicit false overrides.
SELECT f.pk, f.id, f.slug, f.description, f.default_value,
    f.allow_opt_in, f.allow_opt_out, o.value AS override_value
FROM flags f
LEFT JOIN workspace_flag_overrides o
    ON o.flag_id = f.id AND o.workspace_id = sqlc.arg(workspace_id)
ORDER BY f.slug;
