-- name: InsertCustomDomain :exec
INSERT INTO custom_domains (
    id,
    workspace_id,
    project_id,
    app_id,
    environment_id,
    domain,
    challenge_type,
    verification_status,
    verification_token,
    ownership_verified,
    cname_verified,
    target_cname,
    verification_error,
    last_checked_at,
    created_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(project_id),
    sqlc.arg(app_id),
    sqlc.arg(environment_id),
    sqlc.arg(domain),
    sqlc.arg(challenge_type),
    sqlc.arg(verification_status),
    sqlc.arg(verification_token),
    sqlc.arg(ownership_verified),
    sqlc.arg(cname_verified),
    sqlc.arg(target_cname),
    sqlc.arg(verification_error),
    sqlc.arg(last_checked_at),
    sqlc.arg(created_at)
);
