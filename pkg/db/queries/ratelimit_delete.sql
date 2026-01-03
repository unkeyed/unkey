-- name: DeleteRatelimit :exec
DELETE FROM `ratelimits` WHERE id = sqlc.arg('id');