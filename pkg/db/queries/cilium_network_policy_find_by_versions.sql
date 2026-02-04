-- name: ListCiliumNetworkPoliciesByRegion :many
-- ListCiliumNetworkPoliciesByRegion returns cilium network policies for a region with version > after_version.
-- Used by WatchCiliumNetworkPolicies to stream policy state changes to krane agents.
SELECT
    sqlc.embed(n),
    w.k8s_namespace
FROM `cilium_network_policies` n
JOIN `workspaces` w ON w.id = n.workspace_id
WHERE n.region = sqlc.arg(region) AND n.version > sqlc.arg(afterVersion)
ORDER BY n.version ASC
LIMIT ?;
