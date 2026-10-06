-- name: InsertPortalDomain :exec
-- Creates a portal domain awaiting verification. A duplicate (workspace_id, domain)
-- or target_cname surfaces as a unique-key error for the caller to map.
INSERT INTO portal_domains (
    id, workspace_id, portal_id, domain,
    verification_status, verification_token, target_cname,
    domain_connect_provider, domain_connect_url, invocation_id, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
