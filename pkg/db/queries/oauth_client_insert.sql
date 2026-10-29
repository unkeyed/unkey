-- name: InsertOAuthClient :exec
INSERT INTO oauth_clients (
    id,
    name,
    redirect_uris,
    metadata_hash,
    registrar_ip,
    registrar_user_agent,
    created_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(name),
    CAST(sqlc.arg(redirect_uris) AS JSON),
    sqlc.arg(metadata_hash),
    sqlc.arg(registrar_ip),
    sqlc.arg(registrar_user_agent),
    sqlc.arg(created_at)
);
