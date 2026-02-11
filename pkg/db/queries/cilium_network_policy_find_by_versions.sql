-- name: ListCiliumNetworkPoliciesByRegion :many
-- ListCiliumNetworkPoliciesByRegion returns cilium network policies for a region with version > after_version.
-- Used by WatchCiliumNetworkPolicies to stream policy state changes to krane agents.
SELECT *
FROM `cilium_network_policies`
WHERE region = sqlc.arg(region) AND version > sqlc.arg(afterVersion)
ORDER BY version ASC
LIMIT ?;
