-- name: DeleteCiliumNetworkPoliciesByDeploymentIds :exec
-- DeleteCiliumNetworkPoliciesByDeploymentIds removes generated network policy
-- state before its deployments are hard-deleted by the retention sweep.
DELETE FROM cilium_network_policies
WHERE deployment_id IN (sqlc.slice('deployment_ids'));
