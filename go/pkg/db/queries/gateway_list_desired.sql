-- name: ListDesiredGateways :many
SELECT
    *
FROM `gateways`
WHERE (sqlc.arg(region) = '' OR region = sqlc.arg(region))
    AND desired_state = sqlc.arg(desired_state)
    AND id > sqlc.arg(pagination_cursor)
ORDER BY id ASC
LIMIT ?;
