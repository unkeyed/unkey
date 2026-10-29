-- name: InsertRedirectAuthorization :exec
INSERT INTO device_authorizations (
    id,
    user_code,
    expires_at,
    status,
    device_name,
    requester_ip,
    requester_user_agent,
    client_id,
    redirect_uri,
    redirect_uri_presented,
    state,
    code_challenge,
    created_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(user_code),
    sqlc.arg(expires_at),
    'pending',
    sqlc.arg(device_name),
    sqlc.arg(requester_ip),
    sqlc.arg(requester_user_agent),
    sqlc.arg(client_id),
    sqlc.arg(redirect_uri),
    sqlc.arg(redirect_uri_presented),
    sqlc.arg(state),
    sqlc.arg(code_challenge),
    sqlc.arg(created_at)
);
