-- name: ListRatelimitOverridesByNamespaceID :many
SELECT ratelimit_overrides.pk, ratelimit_overrides.id, ratelimit_overrides.workspace_id, ratelimit_overrides.namespace_id, ratelimit_overrides.identifier, ratelimit_overrides.`limit`, ratelimit_overrides.duration, ratelimit_overrides.created_at_m, ratelimit_overrides.updated_at_m, ratelimit_overrides.deleted_at_m FROM ratelimit_overrides
WHERE
workspace_id = sqlc.arg(workspace_id)
AND namespace_id = sqlc.arg(namespace_id)
AND deleted_at_m IS NULL
AND id >= sqlc.arg(cursor_id)
ORDER BY id ASC
LIMIT ?;
