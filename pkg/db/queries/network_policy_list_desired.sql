-- name: ListDesiredNetworkPolicies :many
SELECT
    sqlc.embed(n),
    w.k8s_namespace
FROM `cilium_network_policies` n
INNER JOIN `workspaces` w ON n.workspace_id = w.id
WHERE (sqlc.arg(region) = '' OR n.region = sqlc.arg(region)) AND n.id > sqlc.arg(pagination_cursor)
ORDER BY n.id ASC
LIMIT ?;
