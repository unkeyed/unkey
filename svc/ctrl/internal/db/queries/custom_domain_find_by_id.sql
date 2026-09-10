-- name: FindCustomDomainById :one
SELECT custom_domains.pk, custom_domains.id, custom_domains.workspace_id, custom_domains.project_id, custom_domains.app_id, custom_domains.environment_id, custom_domains.domain, custom_domains.challenge_type, custom_domains.verification_status, custom_domains.verification_token, custom_domains.ownership_verified, custom_domains.cname_verified, custom_domains.target_cname, custom_domains.last_checked_at, custom_domains.check_attempts, custom_domains.verification_error, custom_domains.domain_connect_provider, custom_domains.domain_connect_url, custom_domains.invocation_id, custom_domains.created_at, custom_domains.updated_at
FROM custom_domains
WHERE id = sqlc.arg(id);
