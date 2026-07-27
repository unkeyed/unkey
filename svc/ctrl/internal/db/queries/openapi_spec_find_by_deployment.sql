-- name: FindOpenApiSpecByDeploymentID :one
SELECT openapi_specs.pk, openapi_specs.id, openapi_specs.workspace_id, openapi_specs.deployment_id, openapi_specs.portal_config_id, openapi_specs.content, openapi_specs.created_at, openapi_specs.updated_at FROM openapi_specs WHERE deployment_id = sqlc.arg(deployment_id);
