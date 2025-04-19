-- name: DeleteManyRatelimitsByIDs :exec
DELETE FROM ratelimits WHERE id IN (sqlc.slice(ids));
