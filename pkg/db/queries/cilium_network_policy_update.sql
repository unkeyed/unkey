-- name: UpdateCiliumNetworkPolicyByEnvironmentRegionAndName :exec
UPDATE cilium_network_policies
SET policy = sqlc.arg(policy),
    version = sqlc.arg(version),
    updated_at = sqlc.arg(updated_at)
WHERE environment_id = sqlc.arg(environment_id)
  AND region = sqlc.arg(region)
  AND k8s_name = sqlc.arg(k8s_name);
