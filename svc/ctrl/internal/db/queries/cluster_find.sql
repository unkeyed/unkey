-- name: FindCluster :one
-- FindCluster resolves the cluster and region rows for the complete identity
-- supplied by Krane on cluster-scoped RPCs.
SELECT
    c.pk AS cluster_pk,
    c.id AS cluster_id,
    c.cell_id AS cluster_cell_id,
    c.region_id AS cluster_region_id,
    c.last_heartbeat_at AS cluster_last_heartbeat_at,
    r.pk AS region_pk,
    r.id AS region_id,
    r.name AS region_name,
    r.platform AS region_platform,
    r.can_schedule AS region_can_schedule
FROM clusters c
INNER JOIN regions r ON r.id = c.region_id
WHERE c.cell_id = sqlc.arg(cell_id)
    AND r.platform = sqlc.arg(platform)
    AND r.name = sqlc.arg(region)
LIMIT 1;
