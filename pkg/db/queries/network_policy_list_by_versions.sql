-- name: ListNetworkPolicyByRegion :many
SELECT *
FROM `cilium_network_policies`
WHERE region_id = sqlc.arg(region_id) AND version > sqlc.arg(afterVersion)
ORDER BY version ASC
LIMIT ?;
