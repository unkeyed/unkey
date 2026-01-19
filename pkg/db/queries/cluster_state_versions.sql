-- name: ListClusterStateVersions :many
-- ListClusterStateVersions returns the next N (version, kind) pairs in global version order.
-- Used to determine which resources to fetch for sync, without loading full row data.
-- The 'kind' discriminator is 'deployment' or 'sentinel'.
SELECT combined.version, combined.kind FROM (
    SELECT dt.version, 'deployment' AS kind
    FROM `deployment_topology` dt
    WHERE dt.region = sqlc.arg(region)
        AND dt.version > sqlc.arg(after_version)
    UNION ALL
    SELECT s.version, 'sentinel' AS kind
    FROM `sentinels` s
    WHERE s.region = sqlc.arg(region)
        AND s.version > sqlc.arg(after_version)
) AS combined
ORDER BY combined.version ASC
LIMIT ?;
