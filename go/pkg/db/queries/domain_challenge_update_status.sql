-- name: UpdateDomainChallengeStatus :exec
UPDATE domain_challenges
SET status = ?, updated_at = ?
WHERE domain_id = ?;
