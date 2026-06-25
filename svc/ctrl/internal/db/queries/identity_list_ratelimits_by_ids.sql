-- name: ListIdentityRatelimitsByIDs :many
SELECT * FROM ratelimits WHERE identity_id IN (sqlc.slice(ids));
