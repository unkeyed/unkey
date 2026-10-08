-- name: ScanDeploymentsForCleanup :many
SELECT d.pk, d.environment_id, CAST((e.id IS NULL) AS SIGNED) AS orphaned
FROM (
    SELECT deployments.pk, deployments.environment_id FROM deployments
    WHERE deployments.pk > sqlc.arg(after_pk)
    ORDER BY deployments.pk LIMIT ?
) d
LEFT JOIN environments e ON e.id = d.environment_id
ORDER BY d.pk;
