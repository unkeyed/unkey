-- name: ScanEnvironmentsForCleanup :many
-- Page before checking parents so healthy rows cannot cause an unbounded scan.
SELECT e.pk, e.id, CAST((a.id IS NULL OR p.id IS NULL) AS SIGNED) AS orphaned
FROM (
    SELECT environments.pk, environments.id, environments.app_id, environments.project_id FROM environments
    WHERE environments.pk > sqlc.arg(after_pk)
    ORDER BY environments.pk LIMIT ?
) e
LEFT JOIN apps a ON a.id = e.app_id
LEFT JOIN projects p ON p.id = e.project_id
ORDER BY e.pk;
