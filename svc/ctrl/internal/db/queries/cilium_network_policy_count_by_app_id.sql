-- name: CountCiliumNetworkPoliciesByAppId :one
-- CountCiliumNetworkPoliciesByAppId counts an app's Cilium network policies. Only tests use this query.
SELECT COUNT(*)
FROM `cilium_network_policies`
WHERE app_id = sqlc.arg(app_id);
