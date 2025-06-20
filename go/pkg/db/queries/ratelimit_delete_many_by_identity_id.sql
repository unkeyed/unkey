-- name: DeleteManyRatelimitsByIdentityID :exec
DELETE FROM ratelimits WHERE identity_id = ?;
