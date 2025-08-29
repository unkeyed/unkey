-- name: UpdateDomainChallengeExpiresAt :exec
UPDATE domain_challenges SET expires_at = ? WHERE domain_id = ?;
