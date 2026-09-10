-- name: FindRatelimitOverrideByID :one
SELECT ratelimit_overrides.pk, ratelimit_overrides.id, ratelimit_overrides.workspace_id, ratelimit_overrides.namespace_id, ratelimit_overrides.identifier, ratelimit_overrides.`limit`, ratelimit_overrides.duration, ratelimit_overrides.created_at_m, ratelimit_overrides.updated_at_m, ratelimit_overrides.deleted_at_m FROM ratelimit_overrides
WHERE
    workspace_id = sqlc.arg(workspace_id)
    AND id = sqlc.arg(override_id);
