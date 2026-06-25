-- name: FindCiliumNetworkPolicyByIDAndRegion :one
SELECT *
FROM `cilium_network_policies`
WHERE region_id = sqlc.arg(region_id) AND id = sqlc.arg(cilium_network_policy_id)
LIMIT 1;
