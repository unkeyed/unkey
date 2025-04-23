-- name: DeleteRatelimitsByIdentityID :exec
DELETE FROM ratelimits WHERE identity_id = ?;
