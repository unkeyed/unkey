-- name: FindCluster :one
-- FindCluster resolves the cluster and region rows for the complete identity
-- supplied by Krane on cluster-scoped RPCs.
-- Cluster identifiers are case-sensitive while region identifiers retain their
-- legacy case-insensitive collation, so the join normalizes the region side.
SELECT
    sqlc.embed(c),
    sqlc.embed(r)
FROM clusters c
INNER JOIN regions r ON r.id COLLATE utf8mb4_0900_as_cs = c.region_id
WHERE c.cell_id = sqlc.arg(cell_id)
    AND r.platform = sqlc.arg(platform)
    AND r.name = sqlc.arg(region)
LIMIT 1;
