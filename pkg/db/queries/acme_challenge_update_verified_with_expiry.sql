-- name: UpdateAcmeChallengeVerifiedWithExpiry :exec
UPDATE acme_challenges
SET status = ?, expires_at = ?, updated_at = ?
WHERE domain_id = ?;