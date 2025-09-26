-- name: ListExecutableChallenges :many
SELECT dc.workspace_id, dc.type, d.domain FROM acme_challenges dc
JOIN domains d ON dc.domain_id = d.id
WHERE (dc.status = 'waiting' OR (dc.status = 'verified' AND dc.expires_at <= DATE_ADD(NOW(), INTERVAL 30 DAY)))
AND dc.type IN (sqlc.slice(verification_types))
ORDER BY d.created_at ASC;
