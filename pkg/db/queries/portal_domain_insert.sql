-- name: InsertPortalDomain :exec
-- Production portal domains are created by ctrl; this exists so API tests can
-- seed rows through sqlc instead of hand-written SQL.
INSERT INTO portal_domains (
    id,
    workspace_id,
    portal_id,
    domain,
    verification_status,
    verification_token,
    target_cname,
    created_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(portal_id),
    sqlc.arg(domain),
    sqlc.arg(verification_status),
    sqlc.arg(verification_token),
    sqlc.arg(target_cname),
    sqlc.arg(created_at)
);
