-- name: UpdateDomainChallengeTryClaiming :exec
UPDATE domain_challenges
SET status = ?, updated_at = ?
WHERE domain_id = ? AND status = 'waiting';
