-- name: ListDeploymentImagesForGC :many
SELECT pk, image
FROM deployments
WHERE pk > sqlc.arg(pagination_cursor)
  AND image IS NOT NULL
ORDER BY pk
LIMIT ?;
