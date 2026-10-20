-- name: ScanEnvironmentsForCleanup :many
SELECT e.pk, e.id, CAST(COALESCE((a.id IS NULL OR p.id IS NULL
    OR e.deleted_at < UNIX_TIMESTAMP(CURRENT_TIMESTAMP(3) - INTERVAL 2 HOUR) * 1000
    OR a.deleted_at < UNIX_TIMESTAMP(CURRENT_TIMESTAMP(3) - INTERVAL 2 HOUR) * 1000
    OR p.deleted_at < UNIX_TIMESTAMP(CURRENT_TIMESTAMP(3) - INTERVAL 2 HOUR) * 1000), FALSE) AS SIGNED) AS orphaned
FROM (
    SELECT environments.pk, environments.id, environments.app_id, environments.project_id, environments.deleted_at FROM environments
    WHERE environments.pk > sqlc.arg(after_pk)
    ORDER BY environments.pk LIMIT ?
) e
LEFT JOIN apps a ON a.id = e.app_id
LEFT JOIN projects p ON p.id = e.project_id
ORDER BY e.pk;
