-- name: UpdateVersionOpenApiSpec :exec
UPDATE versions SET 
    openapi_spec = sqlc.arg(openapi_spec)
WHERE id = sqlc.arg(id);