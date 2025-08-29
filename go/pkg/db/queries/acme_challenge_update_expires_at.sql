-- name: UpdateAcmeChallengeExpiresAt :exec
UPDATE acme_challenges SET expires_at = ? WHERE id = ?;
