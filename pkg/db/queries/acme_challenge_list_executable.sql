-- name: ListExecutableChallenges :many
SELECT dc.workspace_id, dc.challenge_type, d.domain FROM acme_challenges dc
JOIN custom_domains d ON dc.domain_id = d.id
WHERE (dc.status = 'waiting' OR (dc.status = 'verified' AND dc.expires_at <= UNIX_TIMESTAMP(DATE_ADD(NOW(), INTERVAL 30 DAY)) * 1000))
AND dc.challenge_type IN (sqlc.slice(verification_types))
ORDER BY d.created_at ASC;
