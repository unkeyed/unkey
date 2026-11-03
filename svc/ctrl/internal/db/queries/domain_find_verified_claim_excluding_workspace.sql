-- name: FindVerifiedDomainClaimExcludingWorkspace :one
-- Finds another workspace's verified claim on a hostname in custom_domains or
-- portal_domains, so ownership contention spans both tables. source names the
-- table holding the claim, which is where a takeover must revoke it.
SELECT 'custom' AS source, custom_domains.id, custom_domains.workspace_id
FROM custom_domains
WHERE custom_domains.domain = sqlc.arg(domain)
  AND custom_domains.workspace_id != sqlc.arg(workspace_id)
  AND custom_domains.verification_status = 'verified'
UNION ALL
SELECT 'portal' AS source, portal_domains.id, portal_domains.workspace_id
FROM portal_domains
WHERE portal_domains.domain = sqlc.arg(domain)
  AND portal_domains.workspace_id != sqlc.arg(workspace_id)
  AND portal_domains.verification_status = 'verified'
LIMIT 1;
