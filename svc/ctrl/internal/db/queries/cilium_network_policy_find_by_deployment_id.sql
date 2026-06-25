-- name: FindCiliumNetworkPoliciesByDeploymentID :many
SELECT * FROM cilium_network_policies WHERE deployment_id = sqlc.arg(deployment_id);
