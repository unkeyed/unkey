-- name: FindFrontlineRouteForPromotion :many
SELECT
    id,
    project_id,
    environment_id,
    hostname,
    deployment_id,
    sticky,
    created_at,
    updated_at
FROM frontline_routes
WHERE
  environment_id = sqlc.arg(environment_id)
  AND sticky IN (sqlc.slice(sticky))
ORDER BY created_at ASC;
