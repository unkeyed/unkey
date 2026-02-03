-- name: InsertCustomDomain :exec
INSERT INTO custom_domains (
    id, workspace_id, project_id, environment_id, domain,
    challenge_type, verification_status, verification_token, target_cname, invocation_id, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
