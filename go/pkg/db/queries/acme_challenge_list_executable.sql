-- name: ListExecutableChallenges :many
SELECT dc.id, dc.workspace_id, domain FROM acme_challenges dc
JOIN domains d ON dc.domain_id = d.id
WHERE dc.status = 'waiting' OR (dc.status = 'verified' AND dc.expires_at <= DATE_ADD(NOW(), INTERVAL 30 DAY))
ORDER BY d.created_at ASC;
