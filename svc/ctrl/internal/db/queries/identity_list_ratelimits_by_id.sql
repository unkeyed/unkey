-- name: ListIdentityRatelimitsByID :many
SELECT * FROM ratelimits WHERE identity_id = sqlc.arg(identity_id)
