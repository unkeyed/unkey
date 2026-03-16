-- name: FindOpenApiSpecByDeploymentID :one
SELECT * FROM openapi_specs WHERE deployment_id = sqlc.arg(deployment_id);
