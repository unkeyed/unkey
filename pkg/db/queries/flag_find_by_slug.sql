-- name: FindFlagBySlug :one
-- FindFlagBySlug locks the definition during enrollment so a concurrent policy
-- change cannot race the permission check and override write. The workspace
-- override is read under the same lock for the audit's effective before value.
SELECT f.pk, f.slug, f.description, f.default_value,
    f.allow_enable, f.allow_disable, o.value AS override_value
FROM flags f
LEFT JOIN flag_overrides o
    ON o.flag_slug = f.slug AND o.workspace_id = sqlc.arg(workspace_id)
WHERE f.slug = sqlc.arg(slug) FOR UPDATE;
