-- name: ListExecutableChallenges :many
-- Lists challenges waiting for issuance or within 30 days of expiry, for deploy
-- and portal domains alike. Only verified domain rows qualify, so a challenge
-- left behind by a failed or revoked domain is never issued.
SELECT dc.workspace_id, dc.challenge_type, d.domain
FROM acme_challenges dc
JOIN (
    SELECT custom_domains.id, custom_domains.domain, custom_domains.created_at
    FROM custom_domains
    WHERE custom_domains.verification_status = 'verified'
    UNION ALL
    SELECT portal_domains.id, portal_domains.domain, portal_domains.created_at
    FROM portal_domains
    WHERE portal_domains.verification_status = 'verified'
) AS d ON dc.domain_id = d.id
WHERE (dc.status = 'waiting' OR (dc.status = 'verified' AND dc.expires_at <= UNIX_TIMESTAMP(DATE_ADD(NOW(), INTERVAL 30 DAY)) * 1000))
AND dc.challenge_type IN (sqlc.slice(verification_types))
ORDER BY d.created_at ASC;
