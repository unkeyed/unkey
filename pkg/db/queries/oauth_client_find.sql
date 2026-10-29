-- name: FindOAuthClientByID :one
SELECT
    pk,
    id,
    name,
    redirect_uris,
    metadata_hash,
    registrar_ip,
    registrar_user_agent,
    created_at
FROM oauth_clients
WHERE id = sqlc.arg(id);

-- name: FindOAuthClientByMetadataHash :one
SELECT
    pk,
    id,
    name,
    redirect_uris,
    metadata_hash,
    registrar_ip,
    registrar_user_agent,
    created_at
FROM oauth_clients
WHERE metadata_hash = sqlc.arg(metadata_hash);
