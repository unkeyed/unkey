-- name: FindCiliumNetworkPolicyByIDAndRegion :one
SELECT *
FROM `cilium_network_policies`
WHERE region = sqlc.arg(region) AND id = sqlc.arg(cilium_network_policy_id)
LIMIT 1;
