-- name: ListWaitingChallenges :many
SELECT dc.id, dc.workspace_id, domain FROM domain_challenges dc
JOIN domains d ON dc.domain_id = d.id
WHERE dc.status = 'waiting' ORDER BY d.created_at ASC;
