-- name: ListPortalDomainsByPortal :many
-- Returns every domain attached to a portal, verified or not. Unscoped by
-- workspace because portal ids are globally unique and ctrl callers already hold one.
SELECT portal_domains.pk, portal_domains.id, portal_domains.workspace_id, portal_domains.portal_id, portal_domains.domain, portal_domains.verification_status, portal_domains.verification_token, portal_domains.ownership_verified, portal_domains.cname_verified, portal_domains.target_cname, portal_domains.last_checked_at, portal_domains.check_attempts, portal_domains.verification_error, portal_domains.domain_connect_provider, portal_domains.domain_connect_url, portal_domains.invocation_id, portal_domains.created_at, portal_domains.updated_at
FROM portal_domains
WHERE portal_id = sqlc.arg(portal_id);
