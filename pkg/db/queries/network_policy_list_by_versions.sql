-- name: ListNetworkPolicyByRegion :many
SELECT
    sqlc.embed(n),
    w.k8s_namespace
FROM `cilium_network_policies` n
INNER JOIN `workspaces` w ON n.workspace_id = w.id
WHERE n.region = sqlc.arg(region) AND n.version > sqlc.arg(afterVersion)
ORDER BY n.version ASC
LIMIT ?;
