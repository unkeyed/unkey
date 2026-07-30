-- name: FindCluster :one
-- FindCluster resolves the cluster and region rows for the complete identity
-- supplied by Krane on cluster-scoped RPCs.
SELECT
    sqlc.embed(c),
    sqlc.embed(r)
FROM clusters c
INNER JOIN regions r ON r.id = c.region_id
WHERE c.cell_id = sqlc.arg(cell_id)
    AND r.platform = sqlc.arg(platform)
    AND r.name = sqlc.arg(region)
LIMIT 1;
