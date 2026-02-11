-- name: ListNetworkPolicyByRegion :many
SELECT *
FROM `cilium_network_policies`
WHERE region = sqlc.arg(region) AND version > sqlc.arg(afterVersion)
ORDER BY version ASC
LIMIT ?;
