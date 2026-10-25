-- name: FindCustomDomainByDomain :one
-- A hostname can have one row per workspace, because uniqueness is
-- (workspace_id, domain): claiming a name does not remove the previous
-- owner's row, so an unverified squatter and the verified owner can both
-- hold it. The caller has no workspace in hand (ACME validation arrives with
-- only a hostname), so pick the verified row and the newest one, which is the
-- owner the contention revoke left in place. Without this the query returned
-- whichever row the optimizer reached first, and certificate issuance
-- encrypted the key under a workspace that no longer owned the domain.
SELECT custom_domains.pk, custom_domains.id, custom_domains.workspace_id, custom_domains.project_id, custom_domains.app_id, custom_domains.environment_id, custom_domains.domain, custom_domains.challenge_type, custom_domains.verification_status, custom_domains.verification_token, custom_domains.ownership_verified, custom_domains.cname_verified, custom_domains.target_cname, custom_domains.last_checked_at, custom_domains.check_attempts, custom_domains.verification_error, custom_domains.domain_connect_provider, custom_domains.domain_connect_url, custom_domains.invocation_id, custom_domains.created_at, custom_domains.updated_at
FROM custom_domains
WHERE domain = sqlc.arg(domain)
ORDER BY (verification_status = 'verified') DESC, created_at DESC
LIMIT 1;
