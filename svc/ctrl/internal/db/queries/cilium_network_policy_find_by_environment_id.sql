-- name: FindCiliumNetworkPoliciesByEnvironmentID :many
SELECT * FROM cilium_network_policies WHERE environment_id = sqlc.arg(environment_id);
