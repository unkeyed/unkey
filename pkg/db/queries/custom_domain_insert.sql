-- name: InsertCustomDomain :exec
INSERT INTO custom_domains (
    id, workspace_id, project_id, app_id, environment_id, domain,
    challenge_type, verification_status, verification_token, target_cname,
    domain_connect_provider, domain_connect_url, invocation_id, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
