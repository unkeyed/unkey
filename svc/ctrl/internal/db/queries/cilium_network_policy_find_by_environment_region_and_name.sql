-- name: FindCiliumNetworkPolicyByEnvironmentRegionAndName :one
SELECT *
FROM `cilium_network_policies`
WHERE environment_id = sqlc.arg(environment_id) AND region_id = sqlc.arg(region_id) AND k8s_name = sqlc.arg(k8s_name)
LIMIT 1;
