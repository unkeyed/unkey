-- name: DeleteCiliumNetworkPoliciesByAppId :exec
DELETE FROM cilium_network_policies WHERE app_id = sqlc.arg(app_id);
