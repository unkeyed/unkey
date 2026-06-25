-- name: ListDesiredNetworkPolicies :many
SELECT *
FROM `cilium_network_policies`
WHERE (sqlc.arg(region_id) = '' OR region_id = sqlc.arg(region_id)) AND id > sqlc.arg(pagination_cursor)
ORDER BY id ASC
LIMIT ?;
