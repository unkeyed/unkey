-- name: ListAllCiliumNetworkPoliciesByRegion :many
-- ListAllCiliumNetworkPoliciesByRegion returns cilium network policies for a region, paginated by pk.
-- Used during full sync (version=0) to bootstrap krane agents with current state.
SELECT * FROM `cilium_network_policies`
WHERE region_id = sqlc.arg(region_id) AND pk > sqlc.arg(after_pk)
ORDER BY pk ASC
LIMIT ?;
