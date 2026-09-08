-- name: ListIdentityRatelimits :many
SELECT ratelimits.pk, ratelimits.id, ratelimits.name, ratelimits.workspace_id, ratelimits.created_at, ratelimits.updated_at, ratelimits.key_id, ratelimits.identity_id, ratelimits.`limit`, ratelimits.duration, ratelimits.auto_apply
FROM ratelimits
WHERE identity_id = sqlc.arg(identity_id)
ORDER BY id ASC
;
