-- name: FilterExistingDepotProjectIds :many
-- FilterExistingDepotProjectIds returns Depot project IDs still referenced by
-- MySQL. Registry reconciliation compares exact IDs so duplicate same-name
-- Depot projects do not survive merely because an Unkey project exists.
SELECT depot_project_id
FROM projects
WHERE depot_project_id IS NOT NULL
  AND depot_project_id IN (sqlc.slice('depot_project_ids'));
