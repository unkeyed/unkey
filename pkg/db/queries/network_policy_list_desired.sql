-- name: ListDesiredNetworkPolicies :many
SELECT *
FROM `cilium_network_policies`
WHERE (sqlc.arg(region) = '' OR region = sqlc.arg(region)) AND id > sqlc.arg(pagination_cursor)
ORDER BY id ASC
LIMIT ?;
