-- name: ListRatelimitOverridesByNamespaceID :many
SELECT * FROM ratelimit_overrides
WHERE
workspace_id = sqlc.arg(workspace_id)
AND namespace_id = sqlc.arg(namespace_id)
AND deleted_at_m IS NULL
AND id >= sqlc.arg(cursor_id)
ORDER BY id ASC
LIMIT ?;
