-- name: FindFlagBySlug :one
-- FindFlagBySlug locks the definition during enrollment so a concurrent policy
-- change cannot race the permission check and override write. The workspace
-- override is read under the same lock for the audit's effective before value.
SELECT f.pk, f.slug, f.description, f.default_enabled,
    f.allow_opt_in, f.allow_opt_out, o.enabled AS override_enabled
FROM flags f
LEFT JOIN flag_overrides o
    ON o.flag_slug = f.slug AND o.workspace_id = sqlc.arg(workspace_id)
WHERE f.slug = sqlc.arg(slug) FOR UPDATE;
