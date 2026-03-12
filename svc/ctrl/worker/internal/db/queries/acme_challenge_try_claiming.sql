-- name: UpdateAcmeChallengeTryClaiming :exec
UPDATE acme_challenges
SET status = ?, updated_at = ?
WHERE domain_id = ? AND status = 'waiting';
